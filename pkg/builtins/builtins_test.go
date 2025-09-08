package builtins_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/pkg/builtins"
)

// Can't be much faster than this..
// BenchmarkRegalLast-10    163252460    7.218 ns/op    0 B/op    0 allocs/op
// ...
func BenchmarkRegalLast(b *testing.B) {
	bctx := rego.BuiltinContext{}
	ta, tb, tc := ast.StringTerm("a"), ast.StringTerm("b"), ast.StringTerm("c")
	arr := ast.ArrayTerm(ta, tb, tc)

	var res *ast.Term

	for b.Loop() {
		var err error
		if res, err = builtins.RegalLast(bctx, arr); err != nil {
			b.Fatal(err)
		}
	}

	if res.Value.Compare(tc.Value) != 0 {
		b.Fatalf("expected c, got %v", res)
	}
}

// Likewise for the empty array case.
// BenchmarkRegalLastEmptyArr-10    160589398    7.498 ns/op    0 B/op    0 allocs/op
// ...
func BenchmarkRegalLastEmptyArr(b *testing.B) {
	bctx := rego.BuiltinContext{}
	arr := ast.ArrayTerm()

	var err error

	for b.Loop() {
		_, err = builtins.RegalLast(bctx, arr)
	}

	if err == nil {
		b.Fatal("expected error, got nil")
	}
}
