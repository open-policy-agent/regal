package encoding

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
)

// Simple routine check to see that things are working as expected.
// While it would be good to add tests for the encoding of each AST node individually, this is thoroughly tested
// via Regal as it consumes the Roast format extensively.

func TestJsonLocationEncoding(t *testing.T) {
	t.Parallel()

	module, err := ast.ParseModuleWithOpts("p.rego", `
package p

import rego.v1

import data.foo.bar

# METADATA
# description: foo bar went to the bar
allow if true

# regular comment

add(x, y) := x + y

partial[x] contains y if {
	some x, y in input

	every z in x {
		z == y
	}
}

obj := {"foo": {"number": 1}, "string": {"set"}, "bool": false}

arr := [1, {"foo": {"key": 1}}]

sc := {x | x := [1, 2, 3][_]}

ac := [x | x := [1, 2, 3][_]]

oc := {k:v | some k, v in input}

test_foo if {
	allow with input as {"foo": "bar"}
}
	`, ast.ParserOptions{ProcessAnnotation: true})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = JSON().Marshal(module); err != nil {
		t.Fatal(err)
	}
}

// https://github.com/open-policy-agent/regal/issues/1592
func TestJSONRoundTripBigNumber(t *testing.T) {
	t.Parallel()

	module := ast.MustParseModule("package p\n\nn := 1e400")

	var modMap map[string]any
	if err := JSONRoundTrip(module, &modMap); err != nil {
		t.Fatalf("failed to marshal module: %v", err)
	}
}
