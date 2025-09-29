package rego

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	emptyResult           = rego.Result{}
	errNoResults          = errors.New("no results returned from evaluation")
	errExcpectedOneResult = errors.New("expected exactly one result from evaluation")
	errExcpectedOneExpr   = errors.New("expected exactly one expression in result")
)

// init [storage.Store] initializes the storage store with the built-in queries.
func init() {
	ast.InternStringTerm(
		// All keys from Code Actions
		"identifier", "workspace_root_uri", "web_server_base_uri", "client", "params", "start", "end",
		"textDocument", "context", "range", "uri", "diagnostics", "only", "triggerKind", "codeDescription",
		"message", "severity", "source", "code", "data", "title", "command", "kind", "isPreferred",
	)
}

type (
	BuiltInCall struct {
		Builtin  *ast.Builtin
		Location *ast.Location
		Args     []*ast.Term
	}

	KeywordUse struct {
		Name     string             `json:"name"`
		Location KeywordUseLocation `json:"location"`
	}

	RuleHeads map[string][]*ast.Location

	KeywordUseLocation struct {
		Row uint `json:"row"`
		Col uint `json:"col"`
	}

	File struct {
		Name                 string             `json:"name"`
		Content              string             `json:"content"`
		Lines                []string           `json:"lines"`
		Abs                  string             `json:"abs"`
		RegoVersion          string             `json:"rego_version"`
		SuccessfulParseCount uint               `json:"successful_parse_count"`
		ParseErrors          []types.Diagnostic `json:"parse_errors"`

		// This exists only for compatibility with some rules in the AST package,
		// where we can't reference e.g. input.params.textDocument.uri without violating
		// the input schema. We should find a better solution for this long-term.
		URI string `json:"uri"`
	}

	Environment struct {
		PathSeparator     string    `json:"path_separator"`
		WorkspaceRootURI  string    `json:"workspace_root_uri"`
		WorkspaceRootPath string    `json:"workspace_root_path"`
		WebServerBaseURI  string    `json:"web_server_base_uri"`
		InputDotJSON      ast.Value `json:"input_dot_json,omitempty"`
		InputDotJSONPath  *string   `json:"input_dot_json_path,omitempty"`
	}

	RegalContext struct {
		Client      types.Client `json:"client"`
		File        File         `json:"file"`
		Environment Environment  `json:"environment"`

		Query *query.Prepared `json:"-"` // for now, might expose to Rego later
	}

	Requirements struct {
		File         FileRequirements `json:"file"`
		InputDotJSON bool             `json:"input_dot_json"`
	}

	FileRequirements struct {
		Lines                    bool `json:"lines"`
		SuccessfulParseLineCount bool `json:"successful_parse_line_count"`
		ParseErrors              bool `json:"parse_errors"`
	}

	Input[T any] struct {
		Method string       `json:"method"`
		Params T            `json:"params"`
		Regal  RegalContext `json:"regal"`
	}

	Result[R any] struct {
		Response R   `json:"response"`
		Regal    any `json:"regal"`
	}

	policy struct {
		module   *ast.Module
		fileName string
		contents string
	}
)

func NewInput[T any](method string, regal RegalContext, params T) Input[T] {
	return Input[T]{Method: method, Regal: regal, Params: params}
}

func (c Input[T]) String() string { // For debugging only
	s, err := encoding.JSON().MarshalToString(&c)
	if err != nil {
		return fmt.Sprintf("Input marshalling error: %v", err)
	}

	return s
}

func PositionFromLocation(loc *ast.Location) types.Position {
	//nolint:gosec
	return types.Position{Line: uint(loc.Row - 1), Character: uint(loc.Col - 1)}
}

func LocationFromPosition(pos types.Position) *ast.Location {
	//nolint: gosec
	return &ast.Location{Row: int(pos.Line + 1), Col: int(pos.Character + 1)}
}

// AllBuiltinCalls returns all built-in calls in the module, excluding operators
// and any other function identified by an infix.
func AllBuiltinCalls(module *ast.Module, builtins map[string]*ast.Builtin) []BuiltInCall {
	builtinCalls := make([]BuiltInCall, 0)

	callVisitor := ast.NewGenericVisitor(func(x any) bool {
		var terms []*ast.Term

		switch node := x.(type) {
		case ast.Call:
			terms = node
		case *ast.Expr:
			if call, ok := node.Terms.([]*ast.Term); ok {
				terms = call
			}
		default:
			return false
		}

		if len(terms) == 0 {
			return false
		}

		if b, ok := builtins[terms[0].Value.String()]; ok {
			// Exclude operators and similar builtins
			if b.Infix != "" {
				return false
			}

			builtinCalls = append(builtinCalls, BuiltInCall{Builtin: b, Location: terms[0].Location, Args: terms[1:]})
		}

		return false
	})

	callVisitor.Walk(module)

	return builtinCalls
}

// AllKeywords returns all keywords in the module.
func AllKeywords(
	ctx context.Context, pq *query.Prepared, fileName, contents string, module *ast.Module,
) (map[string][]KeywordUse, error) {
	var keywords map[string][]KeywordUse

	if err := policyToValue(ctx, pq, policy{module, fileName, contents}, &keywords); err != nil {
		return nil, fmt.Errorf("failed querying for all keywords: %w", err)
	}

	return keywords, nil
}

// AllRuleHeadLocations returns mapping of rules names to the head locations.
func AllRuleHeadLocations(
	ctx context.Context, pq *query.Prepared, fileName, contents string, module *ast.Module,
) (RuleHeads, error) {
	var locations RuleHeads

	err := policyToValue(ctx, pq, policy{module, fileName, contents}, &locations)
	if err != nil {
		return nil, fmt.Errorf("failed querying for rule head locations: %w", err)
	}

	return locations, nil
}

func QueryEval[P any, R any](ctx context.Context, pq *query.Prepared, input Input[P]) (Result[R], error) {
	var result Result[R]

	if err := CachedQueryEval(ctx, pq, rast.StructToValue(input), &result); err != nil {
		return result, fmt.Errorf("failed querying %q: %w", pq, err)
	}

	return result, nil
}

func CachedQueryEval[T any](ctx context.Context, pq *query.Prepared, input ast.Value, toValue *T) error {
	result, err := toValidResult(pq.EvalQuery().Eval(ctx, rego.EvalParsedInput(input)))
	if err != nil {
		return err
	}

	return util.WrapErr(encoding.JSONRoundTrip(result.Expressions[0].Value, toValue), "failed to unmarshal value")
}

func policyToValue[T any](ctx context.Context, pq *query.Prepared, policy policy, toValue *T) error {
	input, err := transform.ToAST(policy.fileName, policy.contents, policy.module, false)
	if err != nil {
		return fmt.Errorf("failed to prepare input: %w", err)
	}

	return CachedQueryEval(ctx, pq, input, toValue)
}

func toValidResult(rs rego.ResultSet, err error) (rego.Result, error) {
	switch {
	case err != nil:
		return emptyResult, fmt.Errorf("evaluation failed: %w", err)
	case len(rs) == 0:
		return emptyResult, errNoResults
	case len(rs) != 1:
		return emptyResult, errExcpectedOneResult
	case len(rs[0].Expressions) != 1:
		return emptyResult, errExcpectedOneExpr
	}

	return rs[0], nil
}
