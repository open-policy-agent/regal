package regal_test

import (
	"errors"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/pkg/builtins/regal"
)

func TestRegalParseModuleWithTemplateString(t *testing.T) {
	t.Parallel()

	policy := `package p

	r := $"{input.foo}"`

	moduleTerm := ast.InternedTerm(policy)
	filenameTerm := ast.InternedTerm("p.rego")
	ops := []*ast.Term{filenameTerm, moduleTerm}

	bctx := rego.BuiltinContext{}

	eqIter := func(term *ast.Term) error {
		result, err := term.Value.Find(ast.Ref{
			ast.InternedTerm("rules"),
			ast.InternedTerm(0),
			ast.InternedTerm("head"),
			ast.InternedTerm("value"),
			ast.InternedTerm("type"),
		})
		if err != nil {
			return err
		}

		if o, ok := result.(ast.String); !ok || string(o) != "templatestring" {
			return errors.New("expected template string type")
		}

		return nil
	}

	if err := regal.RegalParseModule(bctx, ops, eqIter); err != nil {
		t.Fatal(err)
	}
}

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
