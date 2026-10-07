package rego

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	emptyResult           = rego.Result{}
	errNoResults          = errors.New("no results returned from evaluation")
	errExcpectedOneResult = errors.New("expected exactly one result from evaluation")
	errExcpectedOneExpr   = errors.New("expected exactly one expression in result")
)

func init() {
	ast.InternStringTerm(
		// All keys from Code Actions
		"identifier", "workspace_root_uri", "web_server_base_uri", "client", "params", "start", "end",
		"textDocument", "context", "range", "uri", "diagnostics", "only", "triggerKind", "codeDescription",
		"message", "severity", "source", "code", "data", "title", "command", "kind", "isPreferred",
	)
}

type (
	RuleHeads map[string][]*ast.Location

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
		PathSeparator     string `json:"path_separator"`
		WorkspaceRootURI  string `json:"workspace_root_uri"`
		WorkspaceRootPath string `json:"workspace_root_path"`
		InputPath         string `json:"input_path,omitempty"`
	}

	RegalContext struct {
		File        File        `json:"file"`
		Environment Environment `json:"environment"`

		Query *query.Prepared `json:"-"` // for now, might expose to Rego later
	}

	Requirements struct {
		File      FileRequirements `json:"file"`
		InputPath bool             `json:"input_path"`
	}

	FileRequirements struct {
		Lines                    bool `json:"lines"`
		SuccessfulParseLineCount bool `json:"successful_parse_line_count"`
		ParseErrors              bool `json:"parse_errors"`
	}
)

// AllRuleHeadLocations returns mapping of rules names to the head locations.
func AllRuleHeadLocations(
	ctx context.Context, pq *query.Prepared, file, contents string, module *ast.Module,
) (locations RuleHeads, err error) {
	input, err := util.Wrap(transform.ToAST(file, contents, module, false))("failed to prepare input")
	if err == nil {
		err = util.WrapErr(CachedQueryEval(ctx, pq, input, &locations), "failed cached query evaluation")
	}

	return locations, err
}

func CachedQueryEval[T any](ctx context.Context, pq *query.Prepared, input ast.Value, toValue *T) error {
	result, err := toValidResult(pq.EvalQuery().Eval(ctx, rego.EvalParsedInput(input)))
	if err != nil {
		return err
	}

	if val, ok := result.Expressions[0].Value.(ast.Value); ok {
		bs, err := json.Marshal(val, encoding.NoASTOptions)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}

		return util.WrapErr(json.Unmarshal(bs, toValue), "failed to unmarshal value")
	}

	return util.WrapErr(encoding.JSONRoundTrip(result.Expressions[0].Value, toValue), "failed to unmarshal value")
}

func CachedQueryEvalUndecoded[T any](ctx context.Context, pq *query.Prepared, input ast.Value) (res T, err error) {
	result, err := toValidResult(pq.EvalQuery().Eval(ctx, rego.EvalParsedInput(input)))
	if err != nil {
		return res, err
	}

	if res, ok := result.Expressions[0].Value.(T); ok {
		return res, nil
	}

	return res, fmt.Errorf("unexpected query result format: %v", result.Expressions[0].Value)
}

func toValidResult(rs rego.ResultSet, err error) (rego.Result, error) {
	rsLen := len(rs)
	switch {
	case err != nil:
		return emptyResult, fmt.Errorf("evaluation failed: %w", err)
	case rsLen == 0:
		return emptyResult, errNoResults
	case rsLen != 1:
		return emptyResult, errExcpectedOneResult
	case len(rs[0].Expressions) != 1:
		return emptyResult, errExcpectedOneExpr
	}

	return rs[0], nil
}
