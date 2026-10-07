package lsp

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/cover"
	"github.com/open-policy-agent/opa/v1/dependencies"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown/builtins"
	"github.com/open-policy-agent/opa/v1/topdown/print"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	rquery "github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/store"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

var (
	emptyStringAnyMap       = make(map[string]any, 0)
	emptyPrintOutput        = PrintOutput{}
	emptyEvalResult         = EvalResult{}
	workspaceBundleManifest = bundle.Manifest{
		Roots:    &[]string{"workspace"}, // no data in this bundle so no roots are used, however, roots must be set
		Metadata: map[string]any{"name": "workspace"},
	}
	regalEvalUseAsInputComment = regexp.MustCompile(`^\s*regal eval:\s*use-as-input`)
	defaultCaps                = sync.OnceValue(func() *ast.Term {
		return ast.NewTerm(rast.StructToValue(config.CapabilitiesForThisVersion()))
	})
)

type (
	EvalResult struct {
		Value       ast.Value   `json:"value"`
		PrintOutput PrintOutput `json:"printOutput"`
		IsUndefined bool        `json:"isUndefined"`
	}
	PrintOutput map[string]map[int][]string
	PrintHook   struct {
		Output PrintOutput
		// FileNameBase if set, is prepended to filenames in print output. Needed
		// because rego files are evaluated with relative paths (so errors match
		// OPA CLI format) but print hook output consumers need full URIs.
		FileNameBase string
	}
	EvalWorkspaceOptions struct {
		// Query can be an arbitrary selection, so it is not guaranteed to be well-formed
		// Rego.
		Query string
		// Package and Imports let package-relative names and aliases in Query resolve the
		// way they do in the source file they were taken from.
		Package  *ast.Package
		Imports  []*ast.Import
		Coverage *cover.Cover
		RegoOpts []rego.EvalOption
	}
	EvalResponseParams struct {
		Result            EvalResult      `json:"result"`
		Line              int             `json:"line"`
		Target            string          `json:"target"`
		Package           string          `json:"package,omitzero"`              // when target is 'package'
		RuleHeadLocations []*ast.Location `json:"rule_head_locations,omitempty"` // when target is 'rule'
		Coverage          *cover.Report   `json:"coverage,omitempty"`
	}
	prepareOpts struct {
		Body      ast.Body
		Package   *ast.Package
		Imports   []*ast.Import
		Bundles   map[string]*bundle.Bundle
		PrintHook print.Hook
		Store     storage.Store
	}
	prepared struct {
		Query *rego.PreparedEvalQuery
		Opts  prepareOpts
		Args  []func(*rego.Rego)
	}
)

func (l *LanguageServer) handleEvalCommand(ctx context.Context, args types.CommandArgs) (err error) {
	if args.Target == "" || args.Query == "" {
		return fmt.Errorf("expected command target and query, got target %q, query %q", args.Target, args.Query)
	}

	contents, module, ok := l.cache.GetContentAndModule(args.Target)
	if !ok {
		return fmt.Errorf("failed to get content or module for file %q", args.Target)
	}

	var (
		inputValue ast.Value
		inputPath  string
	)

	packagePath := module.Package.Path.String()
	workspace := l.Workspace()
	client := workspace.Client()

	// When the first comment in the file is `regal eval: use-as-input`, the AST of that module is
	// used as the input rather than the contents of input.json/yaml. This is a development feature for
	// working on rules (built-in or custom), allowing querying the AST of the module directly.

	if len(module.Comments) > 0 && regalEvalUseAsInputComment.Match(module.Comments[0].Text) {
		inputValue, err = transform.ToAST(workspace.RelativePath(args.Target), contents, module, false)
		if err != nil {
			return fmt.Errorf("failed to prepare module: %w", err)
		}
	} else {
		// Normal mode — try to find the input.json/yaml file in the workspace and use as input
		// NOTE that we don't break on missing input, as some rules don't depend on that, and should
		// still be evaluable. We may consider returning some notice to the user though.
		inputPath = l.input.FindForPath(args.Target)
		if inputPath == "" && !l.supressInputPrompt {
			ruleName := strings.TrimPrefix(args.Query, packagePath+".")
			created, err := l.handleInputSkeletonPrompt(ctx, args.Target, ruleName, args.Row)
			// Bubbling up an error here if input.json creation fails for any reason.
			if err != nil || created {
				return err
			}
		} else if inputPath != "" {
			inputValue = l.input.Get(ctx, inputPath)
		}
	}

	var (
		result EvalResult
		report *cover.Report
	)

	// This eval sends coverage only if the server and the client both support it, and inline eval too.
	evalWithCoverage := l.featureFlags.InlineEvaluationCoverageProvider &&
		client.InitOptions.EnableEvalInlineCoverage &&
		l.featureFlags.InlineEvaluationProvider &&
		client.InitOptions.EvalCodelensDisplayInline

	workspaceOpts := EvalWorkspaceOptions{Query: args.Query, Package: module.Package, Imports: module.Imports}
	if evalWithCoverage {
		workspaceOpts.Coverage = cover.New()
	}

	if inputValue != nil {
		workspaceOpts.RegoOpts = append(workspaceOpts.RegoOpts, rego.EvalParsedInput(inputValue))
	}

	result, report, err = l.EvalInWorkspace(ctx, workspaceOpts)
	if err != nil {
		return fmt.Errorf("failed to evaluate workspace path: %w", err)
	}

	ruleHeadLocations, err := l.getRuleHeadLocations(ctx, args, contents, module)
	if err != nil {
		return fmt.Errorf("failed to get rule head locations: %w", err)
	}

	target := "package"
	if len(ruleHeadLocations) > 0 {
		target = strings.TrimPrefix(args.Query, packagePath+".")
	}

	if l.featureFlags.InlineEvaluationProvider && client.InitOptions.EvalCodelensDisplayInline {
		responseParams := &EvalResponseParams{
			Result:            result,
			Line:              args.Row,
			Target:            target,
			Package:           strings.TrimPrefix(packagePath, "data."),
			RuleHeadLocations: ruleHeadLocations,
			Coverage:          report,
		}

		bs, err := json.Marshal(responseParams, encoding.NoASTOptions, jsontext.WithIndent("  "))
		if err != nil {
			return fmt.Errorf("failed to marshal eval response params: %w", err)
		}

		// Use a timeout context for RPC to ensure it completes during graceful shutdown
		rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)

		//nolint:contextcheck
		if err = l.conn.Call(rpcCtx, "regal/showEvalResult", new(jsontext.Value(bs)), nil); err != nil {
			l.log.Message("regal/showEvalResult failed: %v", err)
		}

		rpcCancel()
	} else {
		output := workspace.Path("output.json")

		var f *os.File
		if f, err = os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755); err == nil {
			value := result.Value
			if result.IsUndefined {
				value = ast.InternedEmptyObjectValue // undefined displays as an empty object
			}

			err = json.MarshalWrite(f, value, encoding.NoASTOptions, jsontext.WithIndent("  "))

			rio.CloseIgnore(f)
		}
	}

	return err
}

func (l *LanguageServer) getRuleHeadLocations(
	ctx context.Context,
	args types.CommandArgs,
	contents string,
	module *ast.Module,
) ([]*ast.Location, error) {
	pq, err := l.queryCache.GetOrSet(ctx, l.regoStore, query.RuleHeadLocations)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query %s: %w", query.RuleHeadLocations, err)
	}

	file := filepath.Base(uri.ToPath(args.Target))

	allRuleHeadLocations, err := rquery.AllRuleHeadLocations(ctx, pq, file, contents, module)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule head locations: %w", err)
	}

	// if there are none, then it's a package evaluation
	ruleHeadLocations := allRuleHeadLocations[args.Query]

	return ruleHeadLocations, nil
}

func (l *LanguageServer) prepareEval(ctx context.Context, opts EvalWorkspaceOptions) (*prepared, error) {
	body, err := ast.ParseBody(opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed parsing query %q: %w", opts.Query, err)
	}

	if len(body) == 0 {
		// whitespace-only or comment-only selection parses to an empty body.
		return nil, nil //nolint:nilnil
	}

	pops := prepareOpts{
		Body:      body,
		Package:   opts.Package,
		Imports:   opts.Imports,
		Bundles:   l.assembleBundles(),
		PrintHook: PrintHook{Output: PrintOutput{}, FileNameBase: l.Workspace().URI()},
	}

	// We don't want to share the whole store between the server and the
	// eval feature as the latter may expose potentially confusing internals.
	// The config is the same for both though, so avoiding an additional conversion
	// to [ast.Value] is quite desirable.
	confValue, err := store.GetConfig(ctx, l.regoStore)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from store: %w", err)
	}

	conf := ast.NewTerm(confValue)

	pops.Store = inmem.NewFromASTObject(ast.NewObject(
		rast.Item("internal", ast.ObjectTerm(
			rast.Item("combined_config", conf),
			rast.Item("user_config", conf),
			rast.Item("capabilities", rast.GetOr(confValue, "capabilities", defaultCaps)),
		)),
	))

	regoArgs, txn := prepareRegoArgs(ctx, pops)
	defer pops.Store.Abort(ctx, txn)

	// TODO: Let's try to avoid preparing on each eval, but only when the contents
	// of the workspace modules change, and before the user requests an eval.
	pq, err := rego.New(regoArgs...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed preparing query %q: %w", opts.Query, err)
	}

	return &prepared{Query: &pq, Opts: pops, Args: regoArgs}, nil
}

func (l *LanguageServer) EvalInWorkspace(
	ctx context.Context,
	opts EvalWorkspaceOptions,
) (EvalResult, *cover.Report, error) {
	prepared, err := l.prepareEval(ctx, opts)
	if err != nil {
		return emptyEvalResult, nil, err
	}

	if prepared == nil {
		return EvalResult{IsUndefined: true, PrintOutput: emptyPrintOutput}, nil, nil
	}

	var (
		resultSet rego.ResultSet
		ndCache   builtins.NDBCache
	)

	evalOpts := append(slices.Clone(opts.RegoOpts), query.RegoEvalGenJSONValue)

	if opts.Coverage != nil {
		ndCache = builtins.NDBCache{}
		evalOpts = append(evalOpts, rego.EvalQueryTracer(opts.Coverage), rego.EvalNDBuiltinCache(ndCache))
	}

	resultSet, err = prepared.Query.Eval(ctx, evalOpts...)
	if err != nil {
		return emptyEvalResult, nil, fmt.Errorf("failed evaluating query: %w", err)
	}

	hook, ok := prepared.Opts.PrintHook.(PrintHook)
	if !ok {
		hook = PrintHook{Output: PrintOutput{}}
	}

	result := EvalResult{IsUndefined: len(resultSet) == 0, PrintOutput: hook.Output}

	if len(resultSet) > 0 {
		result.Value = finalExpressionValue(prepared.Opts.Body, resultSet[0])
	}

	if opts.Coverage == nil {
		return result, nil, nil
	}

	report, err := l.coverageReport(ctx, prepared.Query, opts.Coverage, ndCache, opts.RegoOpts)
	if err != nil {
		l.log.Message("failed to evaluate coverage for %q, continuing without it: %v", opts.Query, err)

		return result, nil, nil
	}

	return result, report, nil
}

// finalExpressionValue returns the value of the final expression in a query body's result.
//
//   - data.pkg.rule -> the rule's value.
//   - x := 1; y := x + 1; y -> 2, y's bound value.
//   - x := 1; y := x + 1 -> 2, resolved from the := binding, not the raw true.
//   - 1 = x -> 1, resolved from the right-hand side: "=" can bind either side.
//   - [a, b] := [1, 2] -> true, the raw expression value: neither side is a plain
//     variable, so there is no single bound value to return.
func finalExpressionValue(body ast.Body, result rego.Result) ast.Value {
	if len(result.Expressions) == 0 {
		return nil
	}

	last := result.Expressions[len(result.Expressions)-1]

	lastExpr := body[len(body)-1]
	if !lastExpr.IsAssignment() && !lastExpr.IsEquality() {
		return expressionValueToAST(last.Value)
	}

	// ":=" always binds the left operand; "=" (unification) can bind either side.
	operands := []*ast.Term{lastExpr.Operand(0)}
	if lastExpr.IsEquality() {
		operands = append(operands, lastExpr.Operand(1))
	}

	for v := range rast.ValuesOfType[ast.Var](operands) {
		if bound, ok := result.Bindings[string(v)]; ok {
			return expressionValueToAST(bound)
		}
	}

	return expressionValueToAST(last.Value)
}

func expressionValueToAST(expVal any) (val ast.Value) {
	var ok bool
	if val, ok = expVal.(ast.Value); !ok {
		val, _ = ast.InterfaceToValue(expVal)
	}

	return val
}

// coverageReport runs the two supplementary coverage passes (index-excluded, early-exit)
// against the already-prepared query, and merges them into cov's baseline report.
//
// TODO: this hardcodes the two known cover.Kind values. OPA's own tester.Runner
// does the same (opa/v1/tester/runner.go). If OPA adds a new kind, add its
// supplementary pass here too.
func (l *LanguageServer) coverageReport(
	ctx context.Context, pq *rego.PreparedEvalQuery, cov *cover.Cover, ndCache builtins.NDBCache, opts []rego.EvalOption,
) (*cover.Report, error) {
	indexExcluded := cover.New()

	if _, err := pq.Eval(ctx, append(cover.NoIndexingEvalOptions(indexExcluded, ndCache), opts...)...); err != nil {
		return nil, fmt.Errorf("failed evaluating index-excluded coverage query: %w", err)
	}

	earlyExit := cover.New()

	if _, err := pq.Eval(ctx, append(cover.NoEarlyExitEvalOptions(earlyExit, ndCache), opts...)...); err != nil {
		return nil, fmt.Errorf("failed evaluating early-exit coverage query: %w", err)
	}

	cov.AddRun(cover.KindIndexExcluded, indexExcluded)
	cov.AddRun(cover.KindEarlyExit, earlyExit)

	modules := l.cache.GetAllModules()
	modulesByPath := make(map[string]*ast.Module, len(modules))

	for fileURI, module := range modules {
		modulesByPath[l.Workspace().RelativePath(fileURI)] = module
	}

	report := cov.Report(modulesByPath)

	return &report, nil
}

func (l *LanguageServer) debugArgsAssembler(q ast.Body) []func(*rego.Rego) {
	args, _ := prepareRegoArgs(context.TODO(), prepareOpts{
		Body:      q,
		Bundles:   l.assembleBundles(),
		PrintHook: query.StderrPrintHook,
	})

	return args
}

func prepareRegoArgs(ctx context.Context, opts prepareOpts) ([]func(*rego.Rego), storage.Transaction) {
	res := query.SchemaResolvers()
	num := 3 + len(opts.Bundles) + len(res) +
		util.BoolToInt(opts.Package != nil) +
		util.BoolToInt(len(opts.Imports) > 0)

	if opts.Store == nil {
		num += 2
	}

	args := append(make([]func(*rego.Rego), 0, num),
		query.RegoEnablePrint, rego.PrintHook(opts.PrintHook),
		rego.ParsedQuery(opts.Body),
	)

	var txn storage.Transaction
	if opts.Store != nil {
		txn = storage.NewTransactionOrDie(ctx, opts.Store, storage.WriteParams)
		args = append(args, rego.Store(opts.Store), rego.Transaction(txn))
	}

	if opts.Package != nil {
		args = append(args, rego.ParsedPackage(opts.Package))
	}

	if len(opts.Imports) > 0 {
		args = append(args, rego.ParsedImports(opts.Imports))
	}

	for key, b := range opts.Bundles {
		args = append(args, rego.ParsedBundle(key, b))
	}

	return append(args, query.SchemaResolvers()...), txn
}

func (l *LanguageServer) assembleBundles() map[string]*bundle.Bundle {
	// Modules
	modules := l.cache.GetAllModules()
	moduleFiles := make([]bundle.ModuleFile, 0, len(modules))
	hasCustomRules := false

	for fileURI, module := range modules {
		moduleFiles = append(moduleFiles, bundle.ModuleFile{URL: fileURI, Parsed: module, Path: uri.ToPath(fileURI)})
		hasCustomRules = hasCustomRules || strings.Contains(module.Package.Path.String(), "custom.regal.rules")
	}

	// Data
	var dataBundles map[string]bundle.Bundle
	if l.bundleCache != nil {
		dataBundles = l.bundleCache.All()
	}

	allBundles := make(map[string]*bundle.Bundle, len(dataBundles)+2)
	for k := range dataBundles {
		if dataBundles[k].Manifest.Roots != nil {
			allBundles[k] = new(dataBundles[k])
		} else {
			l.log.Message("bundle %s has no roots and will be skipped", k)
		}
	}

	allBundles["workspace"] = &bundle.Bundle{
		Manifest: workspaceBundleManifest,
		Modules:  moduleFiles,
		Data:     emptyStringAnyMap, // Data is sourced from the dataBundles instead
	}

	if hasCustomRules {
		// If someone evaluates a custom Regal rule, provide them the Regal bundle
		// in order to make all Regal functions available
		allBundles["regal"] = rbundle.Loaded()
	}

	return allBundles
}

func (h PrintHook) Print(ctx print.Context, msg string) error {
	filename := ctx.Location.File
	if h.FileNameBase != "" {
		filename = util.EnsureSuffix(h.FileNameBase, "/") + ctx.Location.File
	}

	if _, ok := h.Output[filename]; !ok {
		h.Output[filename] = make(map[int][]string)
	}

	h.Output[filename][ctx.Location.Row] = append(h.Output[filename][ctx.Location.Row], msg)

	return nil
}

func inputSkeletonFromRule(rule *ast.Rule, compiler *ast.Compiler) map[string]any {
	root := map[string]any{}

	refs, err := dependencies.Base(compiler, rule)
	if err != nil {
		return root
	}

	// The logic that resolves dependencies in Base doesn't find refs in the rule head.
	// So, passing that in individually.
	headRefs, err := dependencies.Base(compiler, rule.Head)
	if err != nil {
		return root
	}

	refs = util.Filter(append(refs, headRefs...), func(ref ast.Ref) bool {
		return ref.HasPrefix(ast.InputRootRef) && len(ref) > 1
	})

	for _, ref := range refs {
		node := root

		for _, term := range ref[1 : len(ref)-1] {
			key := strings.Trim(term.Value.String(), `"`)
			// If there's no object for this part of the path, create one
			if _, ok := node[key]; !ok {
				node[key] = map[string]any{}
			}
			// If the object exists, make it the starting point for the next check
			if child, ok := node[key].(map[string]any); ok {
				node = child
			}
		}

		leaf := strings.Trim(ref[len(ref)-1].Value.String(), `"`)
		if _, ok := node[leaf]; !ok {
			node[leaf] = "changeme"
		}
	}

	return root
}
