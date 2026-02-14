package lsp

import (
	"encoding/json"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestPutFileModStoresRoastRepresentation(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()
	fileURI := "file:///example.rego"
	module := parse.MustParseModule("package example\n\nrule := true")

	testutil.NoErr(PutFileMod(t.Context(), store, fileURI, module))(t)

	parsed := must.Return(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "parsed", fileURI}))(t)
	parsedVal := must.Be[ast.Value](t, parsed)
	parsedMap := must.Return(ast.ValueToInterface(parsedVal, nil))(t)
	pretty := must.Return(json.MarshalIndent(parsedMap, "", "  "))(t)

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
	assert.Equal(t, expect, string(pretty))
}

func TestPutFileRefs(t *testing.T) {
	t.Parallel()

	store, fileURI := NewRegalStore(), "file:///example.rego"

	must.Equal(t, nil, PutFileRefs(t.Context(), store, fileURI, []string{"foo", "bar"}))

	val := must.Return(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "defined_refs", fileURI}))(t)
	arr := must.Be[*ast.Array](t, val)
	exp := ast.NewArray(ast.InternedTerm("foo"), ast.InternedTerm("bar"))

	assert.True(t, exp.Equal(arr), "expected defined refs to be %v, got %v", exp, arr)
}

func TestPutBuiltins(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()
	must.Equal(t, nil, PutBuiltins(t.Context(), store, map[string]*ast.Builtin{"count": ast.Count}))

	val := must.Return(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "builtins", "count"}))(t)
	must.NotEqual(t, nil, val, "count builtin should be in store")
}

func TestPutBuiltinsDeprecated(t *testing.T) {
	t.Parallel()

	store := NewRegalStore()
	must.Equal(t, nil, PutBuiltins(t.Context(), store, map[string]*ast.Builtin{"all": ast.All}))

	val := must.Return(storage.ReadOne(t.Context(), store, storage.Path{"workspace", "builtins", "all"}))(t)
	must.NotEqual(t, nil, val, "all builtin should be in store")

	deprecated := must.Be[ast.Object](t, val).Get(ast.StringTerm("deprecated"))
	assert.True(t, ast.Boolean(true).Equal(deprecated.Value), "want deprecated field")
}
