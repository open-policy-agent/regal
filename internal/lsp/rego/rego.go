package rego

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/resolver"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/builtins"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

var (
	emptyResult           = rego.Result{}
	errNoResults          = errors.New("no results returned from evaluation")
	errExcpectedOneResult = errors.New("expected exactly one result from evaluation")
	errExcpectedOneExpr   = errors.New("expected exactly one expression in result")
	prepared              = concurrent.MapOf(make(map[string]*cachedQuery, 5))
	simpleRefPattern      = regexp.MustCompile(`^[a-zA-Z\.]$`)
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
	}

	Environment struct {
		PathSeparator    string `json:"path_separator"`
		WorkspaceRootURI string `json:"workspace_root_uri"`
		WebServerBaseURI string `json:"web_server_base_uri"`
	}

	RegalContext struct {
		Client      types.Client `json:"client"`
		File        File         `json:"file"`
		Environment Environment  `json:"environment"`
	}

	Requirements struct {
		File FileRequirements `json:"file"`
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

	regoOptions = []func(*rego.Rego)

	cachedQuery struct {
		body     ast.Body
		prepared *rego.PreparedEvalQuery
		store    storage.Store
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

type schemaResolver struct {
	value ast.Value
}

func SchemaResolvers() []func(*rego.Rego) {
	return schemaResolvers()
}

func PositionFromLocation(loc *ast.Location) types.Position {
	//nolint:gosec
	return types.Position{Line: uint(loc.Row - 1), Character: uint(loc.Col - 1)}
}

func LocationFromPosition(pos types.Position) *ast.Location {
	//nolint: gosec
	return &ast.Location{Row: int(pos.Line + 1), Col: int(pos.Character + 1)}
}

func ParseQuery(query string) ast.Body {
	if simpleRefPattern.MatchString(query) { // Try cheap parsing if possible
		return rast.RefStringToBody(query)
	}

	return ast.MustParseBody(query)
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
func AllKeywords(ctx context.Context, fileName, contents string, module *ast.Module) (map[string][]KeywordUse, error) {
	var keywords map[string][]KeywordUse

	if err := policyToValue(ctx, query.Keywords, policy{module, fileName, contents}, &keywords); err != nil {
		return nil, fmt.Errorf("failed querying for all keywords: %w", err)
	}

	return keywords, nil
}

// AllRuleHeadLocations returns mapping of rules names to the head locations.
func AllRuleHeadLocations(ctx context.Context, fileName, contents string, module *ast.Module) (RuleHeads, error) {
	var locations RuleHeads

	err := policyToValue(ctx, string(query.RuleHeadLocations), policy{module, fileName, contents}, &locations)
	if err != nil {
		return nil, fmt.Errorf("failed querying for rule head locations: %w", err)
	}

	return locations, nil
}

type Result[R any] struct {
	Response R   `json:"response"`
	Regal    any `json:"regal"`
}

func QueryEval[P any, R any](ctx context.Context, query string, input Input[P]) (Result[R], error) {
	var result Result[R]

	if err := CachedQueryEval(ctx, query, rast.StructToValue(input), &result); err != nil {
		return result, fmt.Errorf("failed querying %q: %w", query, err)
	}

	return result, nil
}

func CachedQueryEval[T any](ctx context.Context, query string, input ast.Value, toValue *T) error {
	cq, err := getOrSetCachedQuery(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed preparing query: %w", err)
	}

	result, err := toValidResult(cq.prepared.Eval(ctx, rego.EvalParsedInput(input)))
	if err != nil {
		return err
	}

	return util.WrapErr(encoding.JSONRoundTrip(result.Expressions[0].Value, toValue), "failed to unmarshal value")
}

func StoreAllCachedQueries(ctx context.Context, store storage.Store) error {
	for _, q := range query.AllQueries() {
		if err := StoreCachedQuery(ctx, q, store); err != nil {
			return fmt.Errorf("failed storing cached query %q: %w", q, err)
		}
	}

	return nil
}

func StoreCachedQuery(ctx context.Context, query string, store storage.Store) error {
	parsedQuery := ParseQuery(query)

	pq, err := prepareQuery(ctx, parsedQuery, store)
	if err != nil {
		return fmt.Errorf("failed preparing query %q: %w", query, err)
	}

	prepared.Set(query, &cachedQuery{body: parsedQuery, prepared: pq, store: store})

	return nil
}

func policyToValue[T any](ctx context.Context, query string, policy policy, toValue *T) error {
	input, err := transform.ToAST(policy.fileName, policy.contents, policy.module, false)
	if err != nil {
		return fmt.Errorf("failed to prepare input: %w", err)
	}

	return CachedQueryEval(ctx, query, input, toValue)
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

func prepareQueryArgs(
	ctx context.Context,
	query ast.Body,
	store storage.Store,
	rb *bundle.Bundle,
) (regoOptions, storage.Transaction) {
	args := make([]func(*rego.Rego), 0, 5+len(builtins.RegalBuiltinRegoFuncs))
	args = append(args, rego.ParsedQuery(query), rego.ParsedBundle("regal", rb))
	args = append(args, builtins.RegalBuiltinRegoFuncs...)

	// For debugging
	args = append(args, rego.EnablePrintStatements(true), rego.PrintHook(topdown.NewPrintHook(os.Stderr)))
	args = append(args, SchemaResolvers()...)

	var txn storage.Transaction
	if store != nil {
		txn, _ = store.NewTransaction(ctx, storage.WriteParams)
		args = append(args, rego.Store(store), rego.Transaction(txn))
	} else {
		args = append(args, rego.StoreReadAST(true))
	}

	return args, txn
}

func getOrSetCachedQuery(ctx context.Context, query string, store storage.Store) (*cachedQuery, error) {
	cq, ok := prepared.Get(query)
	if !ok {
		parsedQuery := ParseQuery(query)

		pq, err := prepareQuery(ctx, parsedQuery, store)
		if err != nil {
			return nil, fmt.Errorf("failed preparing query %q: %w", query, err)
		}

		cq = &cachedQuery{body: parsedQuery, prepared: pq, store: store}

		prepared.Set(query, cq)

		return cq, nil
	}

	if isBundleDevelopmentMode() {
		// In dev mode, we always prepare the query to ensure changes in the bundle are reflected
		// immediately. We can however reuse the query and the store (if set).
		pq, err := prepareQuery(ctx, cq.body, cq.store)
		if err != nil {
			return nil, fmt.Errorf("failed preparing query %q: %w", query, err)
		}

		cq.prepared = pq
	}

	return cq, nil
}

func prepareQuery(ctx context.Context, query ast.Body, store storage.Store) (*rego.PreparedEvalQuery, error) {
	args, txn := prepareQueryArgs(ctx, query, store, rbundle.LoadedBundle())

	// Note that we currently don't provide metrics or profiling here, and
	// most likely we should — need to consider how to best make that conditional
	// and how to present it if enabled.
	pq, err := rego.New(args...).PrepareForEval(ctx)
	if err != nil {
		if store != nil {
			store.Abort(ctx, txn)
		}

		if isBundleDevelopmentMode() {
			// Try falling back to the embedded bundle, or else we'll
			// easily have errors popping up as notifications, making it
			// really hard to fix the issue that broke the query (like a parse error)
			args, txn = prepareQueryArgs(ctx, query, store, rbundle.EmbeddedBundle())
			if pq, err = rego.New(args...).PrepareForEval(ctx); err == nil {
				if store != nil && txn != nil {
					if err = store.Commit(ctx, txn); err != nil {
						return nil, err
					}
				}

				return &pq, nil
			}

			if store != nil {
				store.Abort(ctx, txn)
			}
		}

		return nil, err
	}

	if store != nil && txn != nil {
		if err = store.Commit(ctx, txn); err != nil {
			return nil, err
		}
	}

	return &pq, nil
}

func isBundleDevelopmentMode() bool {
	return os.Getenv("REGAL_BUNDLE_PATH") != ""
}

var schemaResolvers = sync.OnceValue(func() (resolvers []func(*rego.Rego)) {
	ss := compile.RegalSchemaSet()
	added := util.NewSet[string]()

	// Find all schema references in the bundle and add the schemas to the base cache.
	for _, module := range rbundle.LoadedBundle().Modules {
		for _, annos := range module.Parsed.Annotations {
			for _, s := range annos.Schemas {
				if len(s.Schema) == 0 || added.Contains(s.Schema.String()) {
					continue
				}
				resolvers = append(resolvers, rego.Resolver(
					ast.DefaultRootRef.Extend(s.Schema),
					schemaResolver{value: ast.MustInterfaceToValue(ss.Get(s.Schema))},
				))
				added.Add(s.Schema.String())
			}
		}
	}

	return resolvers
})

// Eval implements the resolver.Resolver interface to resolve schemas from annotations at runtime.
func (sr schemaResolver) Eval(_ context.Context, _ resolver.Input) (resolver.Result, error) {
	return resolver.Result{Value: sr.value}, nil
}
