package lsp

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
)

type illegalResolver struct{}

func (illegalResolver) Resolve(ref ast.Ref) (any, error) {
	return nil, fmt.Errorf("illegal value: %v", ref)
}

func TestPutFileModStoresRoastRepresentation(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()
	fileURI := "file:///example.rego"
	module := parse.MustParseModule("package example\n\nrule := true")

	testutil.NoErr(PutFileMod(t.Context(), store, fileURI, module))(t)

	parsed := testutil.Must(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "parsed", fileURI}))(t)
	parsedVal := testutil.MustBe[ast.Value](t, parsed)
	parsedMap := testutil.Must(ast.ValueToInterface(parsedVal, illegalResolver{}))(t)
	pretty := testutil.Must(json.MarshalIndent(parsedMap, "", "  "))(t)

	// This is certainly testing the implementation rather than the behavior, but we actually
	// want some tests to fail if the implementation changes, so we don't have to chase this
	// down elsewhere.
	expect := `{
  "package": {
    "location": "1:1:1:8",
    "path": [
      {
        "type": "var",
        "value": "data"
      },
      {
        "location": "1:9:1:16",
        "type": "string",
        "value": "example"
      }
    ]
  },
  "rules": [
    {
      "head": {
        "assign": true,
        "location": "3:1:3:13",
        "ref": [
          {
            "location": "3:1:3:5",
            "type": "var",
            "value": "rule"
          }
        ],
        "value": {
          "location": "3:9:3:13",
          "type": "boolean",
          "value": true
        }
      },
      "location": "3:1:3:13"
    }
  ]
}`

	if string(pretty) != expect {
		t.Errorf("expected %s, got %s", expect, pretty)
	}
}

func TestPutFileRefs(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()
	fileURI := "file:///example.rego"

	testutil.NoErr(PutFileRefs(t.Context(), store, fileURI, []string{"foo", "bar"}))(t)

	val := testutil.Must(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "defined_refs", fileURI}))(t)
	arr := testutil.MustBe[*ast.Array](t, val)

	if expected := ast.NewArray(ast.StringTerm("foo"), ast.StringTerm("bar")); !expected.Equal(arr) {
		t.Errorf("expected %v, got %v", expected, arr)
	}
}

func TestPutBuiltins(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()

	testutil.NoErr(PutBuiltins(t.Context(), store, map[string]*ast.Builtin{"count": ast.Count}))(t)

	val := testutil.Must(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "builtins", "count"}))(t)
	if val == nil {
		t.Errorf("expected count builtin to exist in store")
	}
}
