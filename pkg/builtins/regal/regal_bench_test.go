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

// 8006 ns/op	   13957 B/op	     242 allocs/op
func BenchmarkRegalParseModule(b *testing.B) {
	bctx := rego.BuiltinContext{}
	policy := "package p\n\nr := \"foo\""
	ops := []*ast.Term{ast.InternedTerm("p.rego"), ast.InternedTerm(policy)}

	for b.Loop() {
		if err := regal.RegalParseModule(bctx, ops, noOpIter); err != nil {
			b.Fatal(err)
		}
	}
}

// This is better tested in OPA — here only to ensure we
// don't accidentally do something wrong on our end.
// See https://github.com/open-policy-agent/opa/pull/8835 for
// one potential improvement to OPA we'd benefit from here
//
// 15567 ns/op	   13572 B/op	     139 allocs/op
func BenchmarkRegalIsFormatted(b *testing.B) {
	bctx := rego.BuiltinContext{}
	policy := "package p\n\nr := \"foo\""
	ops := []*ast.Term{ast.InternedTerm(policy), ast.InternedEmptyObject}

	for b.Loop() {
		if err := regal.RegalIsFormatted(bctx, ops, noOpIter); err != nil {
			b.Fatal(err)
		}
	}
}

func noOpIter(*ast.Term) error { return nil }
