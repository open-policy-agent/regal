package inlayhint_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/inlayhint"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

// A function call may either be represented as an ast.Call.
func TestGetInlayHintsAstCall(t *testing.T) {
	t.Parallel()

	policy := `package p

	r := json.filter({}, [])`

	inlayHints := inlayhint.FromModule(ast.MustParseModule(policy), rego.BuiltinsForDefaultCapabilities())

	must.Equal(t, 2, len(inlayHints), "number of inlay hints")

	assert.Equal(t, "object:", inlayHints[0].Label, "label")
	assert.Equal(t, 2, inlayHints[0].Position.Line, "line")
	assert.Equal(t, 18, inlayHints[0].Position.Character, "character")
	assert.Equal(t, "object to filter\n\nType: `object[any: any]`", inlayHints[0].Tooltip.Value, "tooltip")

	assert.Equal(t, "paths:", inlayHints[1].Label, "label")
	assert.Equal(t, 2, inlayHints[1].Position.Line, "line")
	assert.Equal(t, 22, inlayHints[1].Position.Character, "character")
	assert.Equal(t, "JSON string paths\n\nType: `any<array[any<string, array[any]>], set[any<string, array[any]>]>`",
		inlayHints[1].Tooltip.Value, "tooltip")
}

// Or a function call may be represented as the terms of an ast.Expr.
func TestGetInlayHintsAstTerms(t *testing.T) {
	t.Parallel()

	policy := `package p

	allow if {
		is_string("yes")
	}`

	inlayHints := inlayhint.FromModule(ast.MustParseModule(policy), rego.BuiltinsForDefaultCapabilities())

	must.Equal(t, 1, len(inlayHints), "number of inlay hints")

	assert.Equal(t, "x:", inlayHints[0].Label, "label")
	assert.Equal(t, 3, inlayHints[0].Position.Line, "line")
	assert.Equal(t, 12, inlayHints[0].Position.Character, "character")
	assert.Equal(t, "input value\n\nType: `any`", inlayHints[0].Tooltip.Value, "tooltip")
}
