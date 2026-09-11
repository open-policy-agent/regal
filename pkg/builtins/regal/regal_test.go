package regal_test

import (
	"errors"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/testutil"
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

	testutil.NoErr(regal.RegalParseModule(bctx, ops, eqIter))(t)
}
