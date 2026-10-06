package encoding

import (
	"encoding/json/v2"
	"io"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

var pkg = &ast.Package{
	Location: &ast.Location{Row: 6, Col: 1, Text: []byte("foo")},
	Path:     ast.Ref{ast.DefaultRootDocument, ast.InternedTerm("foo")},
}

func TestAnnotationsOnPackage(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package:     pkg,
		Annotations: []*ast.Annotations{{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"}},
	}

	// package annotations should end up on the package object
	// and *not* on the module object, contrary to how OPA
	// currently does it

	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [
      {
        "type": "var",
        "value": "data"
      },
      {
        "type": "string",
        "value": "foo"
      }
    ],
    "annotations": [
      {
        "location": "1:1:1:1",
        "scope": "package",
        "title": "foo"
      }
    ]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestAnnotationsOnPackageBothPackageAndSubpackagesScope(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{
			{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"},
			{Location: &ast.Location{Row: 3, Col: 1}, Scope: "subpackages", Title: "bar"},
		},
	}

	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [
      {
        "type": "var",
        "value": "data"
      },
      {
        "type": "string",
        "value": "foo"
      }
    ],
    "annotations": [
      {
        "location": "1:1:1:1",
        "scope": "package",
        "title": "foo"
      },
      {
        "location": "3:1:3:1",
        "scope": "subpackages",
        "title": "bar"
      }
    ]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestRuleAndDocumentScopedAnnotationsOnPackageAreDropped(t *testing.T) {
	t.Parallel()

	module := &ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{
			{Location: &ast.Location{Row: 1, Col: 1}, Scope: "package", Title: "foo"},
			{Location: &ast.Location{Row: 3, Col: 1}, Scope: "rule", Title: "bar"},
			{Location: &ast.Location{Row: 4, Col: 1}, Scope: "document", Title: "baz"},
		},
	}

	expected := `{
  "package": {
    "location": "6:1:6:4",
    "path": [
      {
        "type": "var",
        "value": "data"
      },
      {
        "type": "string",
        "value": "foo"
      }
    ],
    "annotations": [
      {
        "location": "1:1:1:1",
        "scope": "package",
        "title": "foo"
      }
    ]
  }
}`
	assert.JSONEqual(t, expected, must.Marshal(t, module, Options))
}

func TestSerializedModuleSize(t *testing.T) {
	t.Parallel()

	policy := mustReadTestFile(t, "testdata/policy.rego")
	module := ast.MustParseModuleWithOpts(string(policy), ast.ParserOptions{ProcessAnnotation: true})

	// This test will fail whenever the size of the serialized module changes,
	// which not often and when it happens it's good to know about it, update
	// and move on.
	must.Equal(t, 79290, len(must.Marshal(t, module, Options)), "serialized module size")
}

// 262180 ns/op      404445 B/op      2648 allocs/op // jsoniter
// 177527 ns/op          50 B/op         1 allocs/op // json/v2
func BenchmarkSerializeModule(b *testing.B) {
	policy := mustReadTestFile(b, "testdata/policy.rego")
	module := ast.MustParseModuleWithOpts(string(policy), ast.ParserOptions{ProcessAnnotation: true})

	var jsonResult []byte

	b.Run("json/v2", func(b *testing.B) {
		opts := json.JoinOptions(Options)
		for b.Loop() {
			if err := json.MarshalWrite(io.Discard, module, opts); err != nil {
				b.Fatalf("failed to marshal module: %v", err)
			}
		}
	})

	_ = jsonResult
}
