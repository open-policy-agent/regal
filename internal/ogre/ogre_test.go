package ogre_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/topdown"

	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	eq        = ast.RefTerm(ast.VarTerm(ast.Equality.Name))
	lintQuery = []*ast.Expr{{ // lint = data.regal.main.lint.
		Terms: []*ast.Term{eq, ast.VarTerm("lint"), ast.RefTerm(
			ast.DefaultRootDocument,
			ast.InternedTerm("regal"),
			ast.InternedTerm("main"),
			ast.InternedTerm("lint"),
		)},
	}}
)

// TestEval tries to mimic a Regal lint evaluation using ogre isolated from the rest of Regal,
// or as much as is possible anyway.
func TestEval(t *testing.T) {
	t.Parallel()

	resultHandler := func(result ast.Value) error {
		violations, ok := rast.GetValue[ast.Set](testutil.MustBe[ast.Object](t, result), "violations")
		if !ok {
			return errors.New("expected violations in result")
		}

		if numViolations := violations.Len(); numViolations != 3 {
			return fmt.Errorf("expected 3 violations, got %d", numViolations)
		}

		return nil
	}

	q := testutil.Must(ogre.New(lintQuery).
		WithPrintHook(topdown.NewPrintHook(t.Output())).
		WithStore(ogre.NewStoreFromObject(t.Context(), mockData(t))).
		Prepare(t.Context()))(t)

	policy := "package foo\n\nx = 1"
	input := testutil.Must(transform.ToAST("p.rego", policy, parse.MustParseModule(policy), false))(t)

	if err := q.Evaluator().WithResultHandler(resultHandler).WithInput(input).Eval(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func mockData(t *testing.T) ast.Object {
	t.Helper()

	conf := testutil.Must(config.FromPath("../../bundle/regal/config/provided/data.yaml"))(t)

	return ast.NewObject(
		rast.Item("eval", ast.ObjectTerm(
			rast.Item("params", ast.ObjectTerm(
				rast.Item("disable_all", ast.InternedTerm(false)),
				rast.Item("disable_category", ast.ArrayTerm()),
				rast.Item("disable", ast.ArrayTerm()),
				rast.Item("enable_all", ast.InternedTerm(true)),
				rast.Item("enable_category", ast.ArrayTerm()),
				rast.Item("enable", ast.ArrayTerm()),
				rast.Item("ignore_files", ast.ArrayTerm()),
			)),
		)),
		rast.Item("internal", ast.ObjectTerm(
			rast.Item("combined_config", ast.NewTerm(conf.ToValue())),
			rast.Item("user_config", ast.ObjectTerm(
				rast.Item("rules", ast.InternedEmptyObject),
			)),
			rast.Item("capabilities", ast.NewTerm(rast.StructToValue(config.CapabilitiesForThisVersion()))),
			rast.Item("path_prefix", ast.InternedTerm("")),
			rast.Item("prepared", ast.ObjectTerm(
				rast.Item("rules_to_run", ast.ObjectTerm(
					rast.Item("idiomatic", ast.SetTerm(
						ast.InternedTerm("directory-package-mismatch"),
					)),
					rast.Item("style", ast.SetTerm(
						ast.InternedTerm("opa-fmt"),
						ast.InternedTerm("use-assignment-operator"),
					)),
				)),
			)),
		)),
	)
}
