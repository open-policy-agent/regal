package evaluate

import (
	"context"
	"errors"
	"fmt"

	godap "github.com/google/go-dap"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/debug"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/funsafe"
	"github.com/open-policy-agent/regal/internal/util"
)

type debugEvaluator struct {
	argsAssembler func(ast.Body) []func(*rego.Rego)
}

// NewHandler returns a new Handler that evaluates expressions in what appears to be the context of a debug session.
// See for example the Delve implementation of this feature to understand why "appears to be" is key here:
//
// https://github.com/go-delve/delve/blob/93f14c7401724729e2127c9aab5b75c5d7dcc152/service/dap/server.go#L3359
//
// Long story short, we can't actually run the eval in a stopped state, since it is... stopped. What we can do is to
// try and reconstruct the state of the program in the given frame, and Rego is pretty good at that given that policies
// don't tend to change the world around them. We'll want to improve this further if we intend to support more complex
// aspects later, like conditional breakpoints on non-trivial expressions.
func NewHandler(argsFn func(ast.Body) []func(*rego.Rego)) Handler {
	return &debugEvaluator{argsAssembler: argsFn}
}

// Evaluate evaluates the given expression in the context of the given debug session and frame.
// See [NewHandler] for more information on how this works.
func (d *debugEvaluator) Evaluate(ctx context.Context, session debug.Session, req Request) (Response, error) {
	if req.Arguments.Expression == "" {
		return NewEmptyResponse(), nil
	}

	vars := varsInScope(session, debug.FrameID(req.Arguments.FrameId), "Locals")

	// If the query is a plain variable name, try to resolve it without eval
	if ast.IsVarCompatibleString(req.Arguments.Expression) {
		for _, v := range vars {
			if v.Name() == req.Arguments.Expression {
				return NewResponse(godap.EvaluateResponseBody{
					Result:             v.Value(),
					Type:               v.Type(),
					VariablesReference: int(v.VariablesReference()),
				}), nil
			}
		}
	}

	expr, err := ast.ParseExpr(req.Arguments.Expression)
	if err != nil {
		return NewEmptyResponse(), err
	}

	query := make(ast.Body, 0, len(vars)+1)
	for _, v := range vars {
		query = append(query, ast.Assign.Expr(
			ast.NewTerm(ast.InternedVarValue(v.Name())),
			ast.NewTerm(funsafe.ToDebugVar(v).ASTValue())),
		)
	}

	query = append(query, expr)

	pq, err := rego.New(d.argsAssembler(query)...).PrepareForEval(ctx)
	if err != nil {
		return NewEmptyResponse(), fmt.Errorf("failed preparing query %s: %w", query, err)
	}

	var evalOpts []rego.EvalOption
	if input := inputFromSession(session, debug.FrameID(req.Arguments.FrameId)); input != nil {
		evalOpts = append(evalOpts, rego.EvalParsedInput(input))
	}

	rs, err := pq.Eval(ctx, evalOpts...)
	if err != nil || len(rs) == 0 {
		return NewEmptyResponse(), err
	}

	if len(rs[0].Expressions) == 0 {
		return NewEmptyResponse(), errors.New("expected at least one expression in result set, got none")
	}

	val := ast.MustInterfaceToValue(rs[0].Expressions[len(rs[0].Expressions)-1].Value)

	return NewResponse(godap.EvaluateResponseBody{Result: val.String(), Type: ast.ValueName(val)}), nil
}

func inputFromSession(session debug.Session, frameID debug.FrameID) ast.Value {
	for _, v := range varsInScope(session, frameID, "Input") {
		if v.Name() == "input" {
			return funsafe.ToDebugVar(v).ASTValue()
		}
	}

	return nil
}

func varsInScope(session debug.Session, frameID debug.FrameID, name string) (vars []debug.Variable) {
	if scopes, err := session.Scopes(frameID); err == nil {
		if scope, ok := util.FindFirst(scopes, func(s debug.Scope) bool { return s.Name() == name }); ok {
			vars, _ = session.Variables(scope.VariablesReference())
		}
	}

	return vars
}
