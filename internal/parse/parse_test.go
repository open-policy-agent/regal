package parse

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestParseModule(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "data.p", must.Return(Module("test.rego", `package p`))(t).Package.Path.String())
}

func TestModuleUnknownVersionWithOpts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		note   string
		policy string
		exp    ast.RegoVersion
		expErr string
	}{
		{
			note: "v1",
			policy: `package p

					 allow if true`,
			exp: ast.RegoV1,
		},
		{
			note: "v1 compatible",
			policy: `package p

					 import rego.v1

					 allow if true`,
			exp: ast.RegoV0CompatV1,
		},
		{
			note: "v0",
			policy: `package p

					 deny["foo"] {
					     true
					 }`,
			exp: ast.RegoV0,
		},
		{
			note:   "unknown / parse error",
			policy: `pecakge p`,
			expErr: "var cannot be used for rule name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.note, func(t *testing.T) {
			t.Parallel()

			parsed, err := ModuleUnknownVersionWithOpts("p.rego", tc.policy, ParserOptions())
			if err != nil {
				if tc.expErr == "" {
					t.Fatalf("unexpected error: %v", err)
				} else {
					testutil.ErrMustContain(err, tc.expErr)(t)
				}
			}

			if parsed == nil && tc.expErr == "" {
				t.Fatal("expected parsed module")
			}

			if tc.expErr == "" && parsed.RegoVersion() != tc.exp {
				t.Errorf("expected version %d, got %d", tc.exp, parsed.RegoVersion())
			}
		})
	}
}
