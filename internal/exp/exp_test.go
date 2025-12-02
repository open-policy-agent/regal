package exp_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

// Cost of defining Rego eval args in-place vs pre-allocating them, benchmarked mostly
// because I was curious about it. While not significant, we may as well pre-allocate
// static values like instrumentation.
//
// in-place_rego_args_-16         54756910        21.34 ns/op      40 B/op       2 allocs/op
// pre-allocated_rego_args_-16    710143035       1.663 ns/op       0 B/op       0 allocs/op
func BenchmarkRegoEvalArgsAlloc(b *testing.B) {
	b.Run("in-place rego args ", func(b *testing.B) {
		input := ast.NewObject()

		for b.Loop() {
			_ = []rego.EvalOption{rego.EvalParsedInput(input), rego.EvalInstrument(true)}
		}
	})

	b.Run("pre-allocated rego args ", func(b *testing.B) {
		input := rego.EvalParsedInput(ast.NewObject())
		instrument := rego.EvalInstrument(true)

		for b.Loop() {
			_ = []rego.EvalOption{input, instrument}
		}
	})
}
