package lsp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rquery "github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
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

func (l *LanguageServer) Eval(
	ctx context.Context, query string, input map[string]any, printHook print.Hook,
) (rego.ResultSet, error) {
	regoArgs := prepareRegoArgs(ast.MustParseBody(query), l.assembleEvalBundles(), printHook, l.getLoadedConfig())

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
		FileNameBase: l.workspaceRootURI,
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
	bundles map[string]bundle.Bundle,
	printHook print.Hook,
	cfg *config.Config,
) []func(*rego.Rego) {
	bundleArgs := make([]func(*rego.Rego), 0, len(bundles))
	for key, b := range bundles { //nolint:gocritic // expensive copy, but I don't think we can avoid it
		bundleArgs = append(bundleArgs, rego.ParsedBundle(key, &b))
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

func (l *LanguageServer) assembleEvalBundles() map[string]bundle.Bundle {
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

	allBundles := make(map[string]bundle.Bundle, len(dataBundles)+2)
	for k := range dataBundles {
		if dataBundles[k].Manifest.Roots != nil {
			allBundles[k] = dataBundles[k]
		} else {
			l.log.Message("bundle %s has no roots and will be skipped", k)
		}
	}

	allBundles["workspace"] = bundle.Bundle{
		Manifest: workspaceBundleManifest,
		Modules:  moduleFiles,
		Data:     emptyStringAnyMap, // Data is sourced from the dataBundles instead
	}

	if hasCustomRules {
		// If someone evaluates a custom Regal rule, provide them the Regal bundle
		// in order to make all Regal functions available
		allBundles["regal"] = *rbundle.Loaded()
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
