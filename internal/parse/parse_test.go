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

func TestParseModuleExperimentalKeywords(t *testing.T) {
	t.Parallel()

	for _, keyword := range []string{"and", "or"} {
		t.Run(keyword, func(t *testing.T) {
			t.Parallel()

			policy := `package p

			import future.keywords.` + keyword + `

			allow if input.a ` + keyword + ` input.b`

			assert.Equal(t, "data.p", must.Return(Module("test.rego", policy))(t).Package.Path.String())
		})
	}
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

			parsed, err := ModuleUnknownVersionWithOpts("p.rego", tc.policy, Options())
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

// ast.ParseBody-12               114549    9125 ns/op    9604 B/op      96 allocs/op
// ast.ParseRef-12                158643    7653 ns/op    7528 B/op      62 allocs/op
// RefStringToBody-12            4431938     269 ns/op     400 B/op      15 allocs/op
// RefStringToRef-12             5975870     201 ns/op     248 B/op      11 allocs/op
// RefStringToBody_interning-12  7036562     169 ns/op     200 B/op       5 allocs/op
// RefStringToRef_interning-12  11741419     103 ns/op      48 B/op       1 allocs/op
func BenchmarkRefStringToBody(b *testing.B) {
	str := "data.foo.bar.baz.qux.quux"
	ref := ast.NewTerm(ast.MustParseRef(str))
	exp := ast.NewBody(ast.NewExpr(ref))

	b.Run("ast.ParseBody", func(b *testing.B) {
		for b.Loop() {
			if body := ast.MustParseBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("ast.ParseRef", func(b *testing.B) {
		for b.Loop() {
			if r := ast.MustParseRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})

	b.Run("RefStringToBody", func(b *testing.B) {
		for b.Loop() {
			if body := RefStringToBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("RefStringToRef", func(b *testing.B) {
		for b.Loop() {
			if r := RefStringToRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})

	ast.InternStringTerm("foo", "bar", "baz", "qux", "quux")

	b.Run("RefStringToBody_interning", func(b *testing.B) {
		for b.Loop() {
			if body := RefStringToBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("RefStringToRef_interning", func(b *testing.B) {
		for b.Loop() {
			if r := RefStringToRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})
}

func TestRefStringToBody(t *testing.T) {
	t.Parallel()

	tests := []string{"data", "data.foo", "data.foo.bar", "input", "input.foo", "var.string1", "var.string1.string2"}

	for _, test := range tests {
		if body := RefStringToBody(test); !body.Equal(ast.MustParseBody(test)) {
			t.Fatalf("expected body to equal %s, got %s", test, body)
		}
	}
}
