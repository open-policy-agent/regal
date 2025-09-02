package documentsymbol_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/documentsymbol"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/types/symbols"
)

func toStrPtr(s string) *string {
	return &s
}

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
					Detail: toStrPtr("single-value rule (number)"),
					Range:  types.RangeBetween(2, 3, 2, 22),
				}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			module, err := ast.ParseModule("test.rego", tc.policy)
			if err != nil {
				t.Fatal(err)
			}

			bis := rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())

			syms := documentsymbol.All(tc.policy, module, bis)

			pkg := syms[0]
			if pkg.Name != tc.expected.Name {
				t.Errorf("Expected %s, got %s", tc.expected.Name, pkg.Name)
			}

			if pkg.Kind != tc.expected.Kind {
				t.Errorf("Expected %d, got %d", tc.expected.Kind, pkg.Kind)
			}

			if pkg.Range != tc.expected.Range {
				t.Errorf("Expected %v, got %v", tc.expected.Range, pkg.Range)
			}

			if pkg.Detail != tc.expected.Detail {
				t.Errorf("Expected %v, got %v", tc.expected.Detail, pkg.Detail)
			}

			if pkg.Children != nil {
				if tc.expected.Children == nil {
					t.Fatalf("Expected no children, got %v", pkg.Children)
				}

				for i, child := range *pkg.Children {
					expectedChild := (*tc.expected.Children)[i]

					if child.Name != expectedChild.Name {
						t.Errorf("Expected %s, got %s", child.Name, expectedChild.Name)
					}

					if child.Kind != expectedChild.Kind {
						t.Errorf("Expected %d, got %d", expectedChild.Kind, child.Kind)
					}

					if child.Range != expectedChild.Range {
						t.Errorf("Expected %v, got %v", expectedChild.Range, child.Range)
					}

					if child.Detail != expectedChild.Detail {
						if expectedChild.Detail == nil && child.Detail != nil {
							t.Errorf("Expected detail to be nilgot %v", child.Detail)
						} else if *child.Detail != *expectedChild.Detail {
							t.Errorf("Expected %s, got %s", *expectedChild.Detail, *child.Detail)
						}
					}
				}
			}
		})
	}
}
