package encoding

import (
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
)

var pkg = &ast.Package{
	Location: &ast.Location{Row: 6, Col: 1, Text: []byte("foo")},
	Path:     ast.Ref{ast.DefaultRootDocument, ast.InternedTerm("foo")},
}

func TestAnnotationsOnPackage(t *testing.T) {
	t.Parallel()

	module := ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{{
			Location: &ast.Location{Row: 1, Col: 1},
			Scope:    "package",
			Title:    "foo",
		}},
	}
	roast := must.Return(jsoniter.ConfigFastest.MarshalIndent(module, "", "  "))(t)

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
	must.Equal(t, expected, string(roast))
}

func TestAnnotationsOnPackageBothPackageAndSubpackagesScope(t *testing.T) {
	t.Parallel()

	module := ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{{
			Location: &ast.Location{Row: 1, Col: 1},
			Scope:    "package",
			Title:    "foo",
		}, {
			Location: &ast.Location{Row: 3, Col: 1},
			Scope:    "subpackages",
			Title:    "bar",
		}},
	}
	roast := must.Return(jsoniter.ConfigFastest.MarshalIndent(module, "", "  "))(t)

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
	must.Equal(t, expected, string(roast))
}

func TestRuleAndDocumentScopedAnnotationsOnPackageAreDropped(t *testing.T) {
	t.Parallel()

	module := ast.Module{
		Package: pkg,
		Annotations: []*ast.Annotations{{
			Location: &ast.Location{Row: 1, Col: 1},
			Scope:    "package",
			Title:    "foo",
		}, {
			Location: &ast.Location{Row: 3, Col: 1},
			Scope:    "rule",
			Title:    "bar",
		}, {
			Location: &ast.Location{Row: 4, Col: 1},
			Scope:    "document",
			Title:    "baz",
		}},
	}
	roast := must.Return(jsoniter.ConfigFastest.MarshalIndent(module, "", "  "))(t)

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
	must.Equal(t, expected, string(roast))
}

func TestSerializedModuleSize(t *testing.T) {
	t.Parallel()

	policy := mustReadTestFile(t, "testdata/policy.rego")
	module := ast.MustParseModuleWithOpts(string(policy), ast.ParserOptions{ProcessAnnotation: true})
	roast := must.Return(jsoniter.ConfigFastest.Marshal(module))(t)

	// This test will fail whenever the size of the serialized module changes,
	// which not often and when it happens it's good to know about it, update
	// and move on.
	must.Equal(t, 85981, len(roast), "serialized module size")
}

// 285329 ns/op	  125555 B/op	    3094 allocs/op
func BenchmarkSerializeModule(b *testing.B) {
	policy := mustReadTestFile(b, "testdata/policy.rego")
	module := ast.MustParseModuleWithOpts(string(policy), ast.ParserOptions{ProcessAnnotation: true})

	for b.Loop() {
		if _, err := jsoniter.ConfigFastest.Marshal(module); err != nil {
			b.Fatalf("failed to marshal module: %v", err)
		}
	}
}
