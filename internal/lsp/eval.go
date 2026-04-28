package lsp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/dependencies"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	rrego "github.com/open-policy-agent/regal/internal/lsp/rego"
	rquery "github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/transform"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

var (
	emptyStringAnyMap       = make(map[string]any, 0)
	emptyEvalResult         = EvalResult{}
	workspaceBundleManifest = bundle.Manifest{
		Roots:    &[]string{"workspace"}, // no data in this bundle so no roots are used, however, roots must be set
		Metadata: map[string]any{"name": "workspace"},
	}
	regalEvalUseAsInputComment = regexp.MustCompile(`^\s*regal eval:\s*use-as-input`)
)

type EvalResult struct {
	Value       any                         `json:"value"`
	PrintOutput map[string]map[int][]string `json:"printOutput"`
	IsUndefined bool                        `json:"isUndefined"`
}

type PrintHook struct {
	Output map[string]map[int][]string
	// FileNameBase if set, is prepended to filenames in print output. Needed
	// because rego files are evaluated with relative paths (so errors match
	// OPA CLI format) but print hook output consumers need full URIs.
	FileNameBase string
}

func (l *LanguageServer) handleEvalCommand(ctx context.Context, args types.CommandArgs) error {
	if args.Target == "" || args.Query == "" {
		l.log.Message("expected command target and query, got target %q, query %q", args.Target, args.Query)

		return nil
	}

	contents, module, ok := l.cache.GetContentAndModule(args.Target)
	if !ok {
		l.log.Message("failed to get content or module for file %q", args.Target)

		return nil
	}

	pq, err := l.queryCache.GetOrSet(ctx, l.regoStore, rquery.RuleHeadLocations)
	if err != nil {
		l.log.Message("failed to prepare query %s", rquery.RuleHeadLocations, err)

		return nil
	}

	file := filepath.Base(uri.ToPath(args.Target))

	allRuleHeadLocations, err := rrego.AllRuleHeadLocations(ctx, pq, file, contents, module)
	if err != nil {
		l.log.Message("failed to get rule head locations: %s", err)

		return nil
	}

	// if there are none, then it's a package evaluation
	ruleHeadLocations := allRuleHeadLocations[args.Query]

	var inputMap map[string]any

	var inputPath string

	// When the first comment in the file is `regal eval: use-as-input`, the AST of that module is
	// used as the input rather than the contents of input.json/yaml. This is a development feature for
	// working on rules (built-in or custom), allowing querying the AST of the module directly.
	if len(module.Comments) > 0 && regalEvalUseAsInputComment.Match(module.Comments[0].Text) {
		//nolint:staticcheck
		inputMap, err = rparse.PrepareAST(l.toRelativePath(args.Target), contents, module)
		if err != nil {
			l.log.Message("failed to prepare module: %s", err)

			return nil
		}
	} else {
		// Normal mode — try to find the input.json/yaml file in the workspace and use as input
		// NOTE that we don't break on missing input, as some rules don't depend on that, and should
		// still be evaluable. We may consider returning some notice to the user though.
		inputPath, inputMap = rio.FindInput(uri.ToPath(args.Target), l.workspacePath())

		ruleName := strings.TrimPrefix(args.Query, module.Package.Path.String()+".")

		if inputPath == "" && !l.supressInputPrompt {
			created, err := l.handleInputSkeletonPrompt(ctx, args.Target, ruleName, args.Row)
			// Bubbling up an error here if input.json creation fails for any reason.
			if err != nil {
				return err
			}

			if created {
				return nil
			}
		}
	}

	var result EvalResult

	if result, err = l.EvalInWorkspace(ctx, args.Query, inputMap); err != nil {
		return fmt.Errorf("failed to evaluate workspace path: %w", err)
	}

	target := "package"
	if len(ruleHeadLocations) > 0 {
		target = strings.TrimPrefix(args.Query, module.Package.Path.String()+".")
	}

	if l.featureFlags.InlineEvaluationProvider && l.getClient().InitOptions.EvalCodelensDisplayInline {
		responseParams := map[string]any{
			"result": result,
			"line":   args.Row,
			"target": target,
			// only used when the target is 'package'
			"package": strings.TrimPrefix(module.Package.Path.String(), "data."),
			// only used when the target is a rule
			"rule_head_locations": ruleHeadLocations,
		}

		responseResult := map[string]any{}

		// Use a timeout context for RPC to ensure it completes during graceful shutdown
		rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)

		//nolint:contextcheck
		if err = l.conn.Call(rpcCtx, "regal/showEvalResult", responseParams, &responseResult); err != nil {
			l.log.Message("regal/showEvalResult failed: %v", err.Error())
		}

		rpcCancel()
	} else {
		output := filepath.Join(l.workspacePath(), "output.json")

		var f *os.File
		if f, err = os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755); err == nil {
			value := result.Value
			if result.IsUndefined {
				value = emptyStringAnyMap // undefined displays as an empty object
			}

			err = encoding.NewIndentEncoder(f, "", "  ").Encode(value)

			rio.CloseIgnore(f)
		}
	}

	return err
}

func (l *LanguageServer) Eval(
	ctx context.Context, query string, input map[string]any, printHook print.Hook,
) (rego.ResultSet, error) {
	regoArgs := prepareRegoArgs(ast.MustParseBody(query), l.assembleBundles(), printHook, l.getLoadedConfig())

	// TODO: Let's try to avoid preparing on each eval, but only when the contents
	// of the workspace modules change, and before the user requests an eval.
	pq, err := rego.New(regoArgs...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed preparing query %s: %w", query, err)
	}

	if input != nil {
		if inputValue, err := transform.ToOPAInputValue(input); err != nil {
			return nil, fmt.Errorf("failed converting input to value: %w", err)
		} else {
			return pq.Eval(ctx, rego.EvalParsedInput(inputValue))
		}
	}

	return pq.Eval(ctx)
}

func (l *LanguageServer) EvalInWorkspace(ctx context.Context, query string, input map[string]any) (EvalResult, error) {
	resultQuery := "result := " + query
	hook := PrintHook{
		Output:       make(map[string]map[int][]string),
		FileNameBase: l.getWorkspaceRootURI(),
	}

	result, err := l.Eval(ctx, resultQuery, input, hook)
	if err != nil {
		return emptyEvalResult, fmt.Errorf("failed evaluating query: %w", err)
	}

	if len(result) == 0 {
		return EvalResult{IsUndefined: true, PrintOutput: hook.Output}, nil
	}

	res, ok := result[0].Bindings["result"]
	if !ok {
		return emptyEvalResult, errors.New("expected result in bindings, didn't get it")
	}

	return EvalResult{Value: res, PrintOutput: hook.Output}, nil
}

func prepareRegoArgs(
	query ast.Body,
	bundles map[string]*bundle.Bundle,
	printHook print.Hook,
	cfg *config.Config,
) []func(*rego.Rego) {
	bundleArgs := make([]func(*rego.Rego), 0, len(bundles))
	for key, b := range bundles {
		bundleArgs = append(bundleArgs, rego.ParsedBundle(key, b))
	}

	schemaResolvers := rquery.SchemaResolvers()
	args := append(
		make([]func(*rego.Rego), 0, 3+len(bundleArgs)+len(schemaResolvers)),
		rego.ParsedQuery(query),
		rego.EnablePrintStatements(true),
		rego.PrintHook(printHook),
	)
	args = append(args, bundleArgs...)
	args = append(args, schemaResolvers...)

	var caps *config.Capabilities
	if cfg != nil && cfg.Capabilities != nil {
		caps = cfg.Capabilities
	} else {
		caps = config.CapabilitiesForThisVersion()
	}

	var evalConfig config.Config
	if cfg != nil {
		evalConfig = *cfg
	}

	userConfigMap := map[string]any{}
	if cfg != nil {
		userConfigMap = config.ToMap(*cfg)
	}

	internalBundle := &bundle.Bundle{
		Manifest: bundle.Manifest{
			Roots:    &[]string{"internal"},
			Metadata: map[string]any{"name": "internal"},
		},
		Data: map[string]any{
			"internal": map[string]any{
				"combined_config": config.ToMap(evalConfig),
				"user_config":     userConfigMap,
				"capabilities":    caps,
			},
		},
	}

	return append(args, rego.ParsedBundle("internal", internalBundle))
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
			b := dataBundles[k]
			allBundles[k] = &b
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

	refs = append(refs, headRefs...)

	for _, ref := range refs {
		// We only want input refs
		if len(ref) < 2 || !ref[0].Equal(ast.InputRootDocument) {
			continue
		}

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
