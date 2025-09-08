package ast

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/parse"
)

func TestGetRuleDetail(t *testing.T) {
	t.Parallel()

	cases := []struct{ input, expected string }{
		{`allow := true`, `single-value rule (boolean)`},
		{`allow := [1,2,3]`, `single-value rule (array)`},
		{`allow := "foo"`, `single-value rule (string)`},
		{`foo contains 1 if true`, `multi-value rule`},
		{`func(x) := true`, `function(x)`},
	}

	bis := rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			mod := parse.MustParseModule("package example\n" + tc.input)
			if len(mod.Rules) != 1 {
				t.Fatalf("Expected 1 rule, got %d", len(mod.Rules))
			}

			if result := GetRuleDetail(mod.Rules[0], bis); result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestSimplifyType(t *testing.T) {
	t.Parallel()

	cases := []struct{ input, expected string }{
		{"set", "set"},
		{"set[any]", "set"},
		{"any<set, object>", "any"},
		{"output: any<set[any], object>", "any"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			if result := simplifyType(tc.input); result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}
