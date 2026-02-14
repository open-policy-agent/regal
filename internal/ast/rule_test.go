package ast

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
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

	bis := rego.BuiltinsForCapabilities(io.Capabilities())

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			mod := parse.MustParseModule("package example\n" + tc.input)

			must.Equal(t, 1, len(mod.Rules), "number of rules in module")
			assert.Equal(t, tc.expected, GetRuleDetail(mod.Rules[0], bis))
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
			assert.Equal(t, tc.expected, simplifyType(tc.input))
		})
	}
}
