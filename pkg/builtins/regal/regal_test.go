package regal_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/pkg/builtins/regal"
)

// Can't be much faster than this..
// BenchmarkRegalLast-10    163252460    7.218 ns/op    0 B/op    0 allocs/op
// ...
func BenchmarkRegalLast(b *testing.B) {
	bctx := rego.BuiltinContext{}
	ta, tb, tc := ast.StringTerm("a"), ast.StringTerm("b"), ast.StringTerm("c")
	arr := ast.ArrayTerm(ta, tb, tc)
	ops := []*ast.Term{arr}

	eqIter := func(t *ast.Term) error {
		if t != tc {
			b.Fatalf("expected %v, got %v", tc, t)
		}

		return nil
	}

	for b.Loop() {
		if err := regal.RegalLast(bctx, ops, eqIter); err != nil {
			b.Fatal(err)
		}
	}
}

// Likewise for the empty array case.
// BenchmarkRegalLastEmptyArr-10    160589398    7.498 ns/op    0 B/op    0 allocs/op
// ...
func BenchmarkRegalLastEmptyArr(b *testing.B) {
	bctx := rego.BuiltinContext{}
	iter := func(t *ast.Term) error {
		if t != nil {
			b.Fatalf("expected nil, got %v", t)
		}

		return nil
	}
	arr := ast.ArrayTerm()
	ops := []*ast.Term{arr}

	for b.Loop() {
		if err := regal.RegalLast(bctx, ops, iter); err != nil {
			b.Fatal(err)
		}
	}
}
