package semantictokens

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/opa/v1/ast"
)

func TestFull(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		policy         string
		expectedTokens []uint
	}{
		"package only": {
			policy: `package regal.woo`,
			expectedTokens: []uint{
				0, 8, 5, 0, 0,
				0, 6, 3, 0, 0,
			},
		},
		"variable declarations": {
			policy: `package regal.woo

test_function(param1, param2) := result if {
      true
}
`,
			expectedTokens: []uint{
				0, 8, 5, 0, 0,
				0, 6, 3, 0, 0,
				2, 14, 6, 1, 1,
				0, 8, 6, 1, 1,
			},
		},
		"variable references": {
			policy: `package regal.woo
			
test_function(param1) := result if {
      calc3 := 1
      calc3 == param1
}
`,
			expectedTokens: []uint{
				0, 8, 5, 0, 0,
				0, 6, 3, 0, 0,
				2, 14, 6, 1, 1,
				2, 15, 6, 1, 2,
			},
		},
		"full policy with package, declarations and references": {
			policy: `package regal.woo
			
test_function(param1) := result if {
	  calc1 := param1 * 2
      calc2 := param2 + 10
      result := calc1 + calc2
	  
      calc3 := 1
      calc3 == param1
}
`,
			expectedTokens: []uint{
				0, 8, 5, 0, 0,
				0, 6, 3, 0, 0,
				2, 14, 6, 1, 1,
				1, 12, 6, 1, 2,
				5, 15, 6, 1, 2,
			},
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			module := ast.MustParseModule(tc.policy)

			result, err := Full(module)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			t.Logf("Actual tokens: %v", result.Data)
			t.Logf("Expected tokens: %v", tc.expectedTokens)

			if diff := cmp.Diff(result.Data, tc.expectedTokens); diff != "" {
				t.Errorf("unexpected token data (-got +want):\n%s", diff)
			}
		})
	}
}
