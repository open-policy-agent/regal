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
		expectedTokens []semanticTokenInstance
	}{
		"package only": {
			policy: `package regal.woo`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
			},
		},
		"variable declarations": {
			policy: `package regal.woo

test_function(param1, param2) := result if {
      true
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 0, DeltaCol: 8, Length: 6, Type: 1, Modifier: 1},
			},
		},
		"variable references": {
			policy: `package regal.woo
			
test_function(param1) := result if {
      calc3 := 1
      calc3 == param1
}
`,
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 2, DeltaCol: 15, Length: 6, Type: 1, Modifier: 2},
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
			expectedTokens: []semanticTokenInstance{
				{DeltaLine: 0, DeltaCol: 8, Length: 5, Type: 0, Modifier: 0},
				{DeltaLine: 0, DeltaCol: 6, Length: 3, Type: 0, Modifier: 0},
				{DeltaLine: 2, DeltaCol: 14, Length: 6, Type: 1, Modifier: 1},
				{DeltaLine: 1, DeltaCol: 12, Length: 6, Type: 1, Modifier: 2},
				{DeltaLine: 5, DeltaCol: 15, Length: 6, Type: 1, Modifier: 2},
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

			// Convert actual result to readable format
			actualTokens := uintsToTestTokens(result.Data)

			t.Logf("Actual tokens: %+v", actualTokens)
			t.Logf("Expected tokens: %+v", tc.expectedTokens)

			if diff := cmp.Diff(actualTokens, tc.expectedTokens); diff != "" {
				t.Errorf("unexpected token data (-got +want):\n%s", diff)
			}
		})
	}
}

// semanticTokenInstance adds structure to uint data stream in the SemanticToken
// return type making it more readable for error messages and comparisons in tests
type semanticTokenInstance struct {
	DeltaLine uint
	DeltaCol  uint
	Length    uint
	Type      uint
	Modifier  uint
}

func uintsToTestTokens(data []uint) []semanticTokenInstance {
	if len(data)%5 != 0 {
		panic("invalid token data length, must be multiple of 5")
	}

	tokens := make([]semanticTokenInstance, 0, len(data)/5)
	for i := 0; i < len(data); i += 5 {
		tokens = append(tokens, semanticTokenInstance{
			DeltaLine: data[i],
			DeltaCol:  data[i+1],
			Length:    data[i+2],
			Type:      data[i+3],
			Modifier:  data[i+4],
		})
	}

	return tokens
}
