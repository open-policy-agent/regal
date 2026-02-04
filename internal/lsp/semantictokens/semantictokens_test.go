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

			result, err := Full(t.Context(), module)
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

func BenchmarkFullOneFunction(b *testing.B) {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	calc1 := param1 * 2
	calc2 := param2 + 10
	result := calc1 + calc2

	calc3 := 1
	calc3 == param1
}`

	module := ast.MustParseModule(policy)

	b.ResetTimer()

	for b.Loop() {
		result, err := Full(b.Context(), module)
		if err != nil {
			b.Fatal(err)
		}

		_ = result
	}
}

func BenchmarkFullTwoFunctions(b *testing.B) {
	policy := `package regal.woo

test_function_one(param1, param2) := result if {
	calc1 := param1 * 2
	calc2 := param2 + 10
	result := calc1 + calc2
}

test_function_two(x, y, z) := output if {
	temp := x + y
	output := temp * z
}`

	module := ast.MustParseModule(policy)

	b.ResetTimer()

	for b.Loop() {
		result, err := Full(b.Context(), module)
		if err != nil {
			b.Fatal(err)
		}

		_ = result
	}
}

func BenchmarkFullFiveFunctions(b *testing.B) {
	policy := `package regal.woo

test_function_one(a, b) := result if {
	temp := a * b
	result := temp + 10
}

test_function_two(x, y, z) := output if {
	calc := x + y
	output := calc * z
}

test_function_three(p1, p2, p3, p4) := value if {
	step1 := p1 + p2
	step2 := p3 - p4
	value := step1 * step2
}

test_function_four(input1, input2) := final if {
	intermediate := input1 / input2
	final := intermediate + 100
}

test_function_five(v1, v2, v3, v4, v5) := combined if {
	sum := v1 + v2 + v3
	product := v4 * v5
	combined := sum + product
}`

	module := ast.MustParseModule(policy)

	b.ResetTimer()

	for b.Loop() {
		result, err := Full(b.Context(), module)
		if err != nil {
			b.Fatal(err)
		}

		_ = result
	}
}

func BenchmarkFullTenFunctions(b *testing.B) {
	policy := `package regal.woo

test_function_one(a, b) := result if {
	result := a + b
}

test_function_two(x, y) := output if {
	output := x * y
}

test_function_three(p, q, r) := value if {
	temp := p + q
	value := temp - r
}

test_function_four(m, n) := result if {
	result := m / n
}

test_function_five(v1, v2, v3) := combined if {
	sum := v1 + v2
	combined := sum * v3
}

test_function_six(param1, param2, param3, param4) := final if {
	step1 := param1 * param2
	step2 := param3 + param4
	final := step1 - step2
}

test_function_seven(input1, input2) := output if {
	temp := input1 + 50
	output := temp * input2
}

test_function_eight(a, b, c, d, e) := result if {
	part1 := a + b + c
	part2 := d * e
	result := part1 + part2
}

test_function_nine(x, y, z) := value if {
	intermediate := x * y
	value := intermediate + z
}

test_function_ten(p1, p2, p3, p4, p5, p6) := final if {
	group1 := p1 + p2 + p3
	group2 := p4 * p5 * p6
	temp := group1 + group2
	final := temp / 2
}`

	module := ast.MustParseModule(policy)

	b.ResetTimer()

	for b.Loop() {
		result, err := Full(b.Context(), module)
		if err != nil {
			b.Fatal(err)
		}

		_ = result
	}
}
