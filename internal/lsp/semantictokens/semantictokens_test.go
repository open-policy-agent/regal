package semantictokens

import (
	"encoding/json"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

func TestFull(t *testing.T) {
	t.Parallel()

	policy := `
# METADATA
# description: Ambiguous metadata scope
package regal.woo

# METADATA
# description: Ambiguous metadata scope
test_function(param1, param2) := result if {
      calc3 := 1
      calc3 == param1
      calc1 := param1 * 2
      calc2 := param2 + 10
      result := calc1 + calc2
}

allow if test_function(1, 2) == 14
`

	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
		ProcessAnnotation: true,
	})

	roastInput, err := transform.ToAST("semantictokens.rego", "", module, true)
	if err != nil {
		t.Fatal("Failed to transform to roast format")
	}

	// Pretty print roast input as JSON
	if jsonBytes, err := json.MarshalIndent(roastInput, "", "  "); err == nil {
		t.Logf("Roast input (JSON):\n%s", string(jsonBytes))
	} else {
		t.Logf("Failed to marshal roast input: %v", err)
		t.Logf("Roast input (raw): %#v", roastInput)
	}

	result, err := Full(module)
	if err != nil {
		t.Logf("%#v", err)
		t.Fatal("Result expected what happened")
	}

	t.Logf("%#v", result)
	// t.Fail()
}

// func TestPackages(t *testing.T) {
// 	t.Parallel()

// 	policy := ``

// 	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
// 		ProcessAnnotation: true,
// 	})

// 	result, err := Full(module)
// 	if err != nil {
// 		t.Logf("%#v", err)
// 		t.Fatal("Result expected what happened")
// 	}

// 	t.Logf("%#v", result)
// 	// t.Fail()
// }

// func TestVarDeclaration(t *testing.T) {
// 	t.Parallel()

// 	policy := ``

// 	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
// 		ProcessAnnotation: true,
// 	})

// 	result, err := Full(module)
// 	if err != nil {
// 		t.Logf("%#v", err)
// 		t.Fatal("Result expected what happened")
// 	}

// 	t.Logf("%#v", result)
// 	// t.Fail()
// }

// func TestVarReference(t *testing.T) {
// 	t.Parallel()

// 	policy := ``

// 	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
// 		ProcessAnnotation: true,
// 	})

// 	result, err := Full(module)
// 	if err != nil {
// 		t.Logf("%#v", err)
// 		t.Fatal("Result expected what happened")
// 	}

// 	t.Logf("%#v", result)
// 	// t.Fail()
// }
