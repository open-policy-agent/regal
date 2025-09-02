package inlayhint_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/inlayhint"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
)

// A function call may either be represented as an ast.Call.
func TestGetInlayHintsAstCall(t *testing.T) {
	t.Parallel()

	policy := `package p

	r := json.filter({}, [])`

	module := ast.MustParseModule(policy)

	bis := rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())
	inlayHints := inlayhint.FromModule(module, bis)

	if len(inlayHints) != 2 {
		t.Fatalf("Expected 2 inlay hints, got %d", len(inlayHints))
	}

	if inlayHints[0].Label != "object:" {
		t.Errorf("Expected label to be 'object:', got %s", inlayHints[0].Label)
	}

	if inlayHints[0].Position.Line != 2 && inlayHints[0].Position.Character != 18 {
		t.Errorf("Expected line 2, character 18, got %d, %d",
			inlayHints[0].Position.Line, inlayHints[0].Position.Character)
	}

	if inlayHints[0].Tooltip.Value != "object to filter\n\nType: `object[any: any]`" {
		t.Errorf("Expected tooltip to be 'object to filter\n\nType: `object[any: any]`, got %s", inlayHints[0].Tooltip.Value)
	}

	if inlayHints[1].Label != "paths:" {
		t.Errorf("Expected label to be 'paths:', got %s", inlayHints[1].Label)
	}

	if inlayHints[1].Position.Line != 2 && inlayHints[1].Position.Character != 22 {
		t.Errorf("Expected line 2, character 22, got %d, %d",
			inlayHints[1].Position.Line, inlayHints[1].Position.Character)
	}

	if inlayHints[1].Tooltip.Value != "JSON string paths\n\nType: `any<array[any<string, array[any]>],"+
		" set[any<string, array[any]>]>`" {
		t.Errorf("Expected tooltip to be 'JSON string paths\n\nType: `any<array[any<string, array[any]>], "+
			"set[any<string, array[any]>]>`, got %s", inlayHints[1].Tooltip.Value)
	}
}

// Or a function call may be represented as the terms of an ast.Expr.
func TestGetInlayHintsAstTerms(t *testing.T) {
	t.Parallel()

	policy := `package p

	allow if {
		is_string("yes")
	}`

	module := ast.MustParseModule(policy)

	bis := rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())

	inlayHints := inlayhint.FromModule(module, bis)

	if len(inlayHints) != 1 {
		t.Fatalf("Expected 1 inlay hints, got %d", len(inlayHints))
	}

	if inlayHints[0].Label != "x:" {
		t.Errorf("Expected label to be 'x:', got %s", inlayHints[0].Label)
	}

	if inlayHints[0].Position.Line != 5 && inlayHints[0].Position.Character != 12 {
		t.Errorf("Expected line 5, character 12, got %d, %d",
			inlayHints[0].Position.Line, inlayHints[0].Position.Character)
	}

	if inlayHints[0].Tooltip.Value != "input value\n\nType: `any`" {
		t.Errorf("Expected tooltip to be 'input value\n\nType: `any`, got %s", inlayHints[0].Tooltip.Value)
	}
}
