package documentsymbol_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/documentsymbol"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/types/symbols"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestDocumentSymbols(t *testing.T) {
	t.Parallel()

	cases := []struct {
		title    string
		policy   string
		expected types.DocumentSymbol
	}{
		{
			"only package",
			`package foo`,
			types.DocumentSymbol{
				Name:  "data.foo",
				Kind:  symbols.Package,
				Range: types.RangeBetween(0, 0, 0, 11),
			},
		},
		{
			"call",
			`package p

			i := indexof("a", "a")`,
			types.DocumentSymbol{
				Name:  "data.p",
				Kind:  symbols.Package,
				Range: types.RangeBetween(0, 0, 2, 25),
				Children: &[]types.DocumentSymbol{{
					Name:   "i",
					Kind:   symbols.Variable,
					Detail: new("single-value rule (number)"),
					Range:  types.RangeBetween(2, 3, 2, 22),
				}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			bis := rego.BuiltinsForDefaultCapabilities()
			syms := documentsymbol.All(tc.policy, ast.MustParseModule(tc.policy), bis)

			pkg := syms[0]

			assert.Equal(t, tc.expected.Name, pkg.Name, "name")
			assert.Equal(t, tc.expected.Kind, pkg.Kind, "kind")
			assert.Equal(t, tc.expected.Range, pkg.Range, "range")
			assert.Equal(t, tc.expected.Detail, pkg.Detail, "detail")

			if pkg.Children != nil {
				must.NotEqual(t, nil, tc.expected.Children, "expected children")

				for i, child := range *pkg.Children {
					expectedChild := (*tc.expected.Children)[i]

					assert.Equal(t, expectedChild.Name, child.Name, "name")
					assert.Equal(t, expectedChild.Kind, child.Kind, "kind")
					assert.Equal(t, expectedChild.Range, child.Range, "range")

					if child.Detail != expectedChild.Detail {
						if expectedChild.Detail == nil && child.Detail != nil {
							t.Errorf("Expected detail to be nil, got %v", child.Detail)
						} else if *child.Detail != *expectedChild.Detail {
							t.Errorf("Expected %s, got %s", *expectedChild.Detail, *child.Detail)
						}
					}
				}
			}
		})
	}
}
