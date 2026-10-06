package lsp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/runtime/info"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/pkg/config"
)

var runtimeInfo = sync.OnceValue(func() *ast.Term {
	info, err := info.New()
	if err != nil {
		info = ast.InternedEmptyObject
	}

	return info
})

// handleRunTests handles the regal/runTests LSP request.
// It runs OPA tests based on the provided parameters and returns results.
func (l *LanguageServer) handleRunTests(ctx context.Context, params types.RunTestsParams) (any, error) {
	// Ensure the target file is parsed before running tests.
	// This handles the case where regal/runTests is called immediately after
	// textDocument/didOpen, before the async diagnostics worker has parsed the file.
	// This makes the code more robust, but is mostly only helpful in tests.
	for _, target := range slices.Concat(params.Targets, params.Exclude) {
		if target.URI == "" {
			continue
		}

		if _, ok := l.cache.GetModule(target.URI); !ok {
			if _, err := updateParse(ctx, l.parseOpts(target.URI, l.builtinsForCurrentCapabilities())); err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", target.URI, err)
			}
		}
	}

	prefixes, err := testPrefixes(l.cache.GetAllModules(), params.Targets, params.Exclude)
	if err != nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
	}

	// If the client has not asked it'll discard results.
	if len(prefixes) == 0 {
		return []tester.Result{}, nil
	}

	store, txn := newStoreAndTxn(ctx, l.getLoadedConfig())

	defer store.Abort(ctx, txn)

	runner := tester.NewRunner().
		SetCompiler(compile.NewCompilerWithRegalBuiltins().
			WithEnablePrintStatements(true).
			WithUseTypeCheckAnnotations(true)).
		SetStore(store).
		SetBundles(l.assembleBundles()).
		SetRuntime(runtimeInfo()).
		CapturePrintOutput(true).
		SetTimeout(5 * time.Second).
		SetPrefixMatchers(prefixes...)

	ch, err := runner.RunTests(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	return collectTestResults(ch), nil
}

func testPrefixes(modules map[string]*ast.Module, targets, exclude []types.TestTarget) ([]ast.Ref, error) {
	if len(targets) == 0 {
		return nil, errors.New("regal/runTests needs at least one target, use {} to run all tests")
	}

	include, err := targetRefs(modules, targets)
	if err != nil {
		return nil, err
	}

	if len(exclude) == 0 {
		return include, nil
	}

	excluded, err := targetRefs(modules, exclude)
	if err != nil {
		return nil, err
	}

	var refs []ast.Ref

	for _, module := range modules {
		for _, ref := range testRefs(module) {
			if slices.ContainsFunc(include, ref.HasPrefix) && !slices.ContainsFunc(excluded, ref.HasPrefix) {
				refs = append(refs, ref)
			}
		}
	}

	return refs, nil
}

func targetRefs(modules map[string]*ast.Module, targets []types.TestTarget) ([]ast.Ref, error) {
	refs := make([]ast.Ref, 0, len(targets))

	for _, target := range targets {
		switch {
		case target.URI != "":
			module, ok := modules[target.URI]
			if !ok {
				return nil, fmt.Errorf("invalid test target: no parsed module for %q", target.URI)
			}

			refs = append(refs, testRefs(module)...)
		case target.Package == "" && target.Name != "":
			return nil, fmt.Errorf("invalid test target: name %q has no package", target.Name)
		case target.Package == "":
			refs = append(refs, ast.DefaultRootRef)
		default:
			ref, err := ast.ParseRef(strings.Join(slices.DeleteFunc([]string{target.Package, target.Name},
				func(s string) bool { return s == "" }), "."))
			if err != nil {
				return nil, fmt.Errorf("invalid test target %q %q: %w", target.Package, target.Name, err)
			}

			if !ref.HasPrefix(ast.DefaultRootRef) {
				return nil, fmt.Errorf("invalid test target %q: package must start with data", target.Package)
			}

			refs = append(refs, ref)
		}
	}

	return refs, nil
}

func testRefs(module *ast.Module) []ast.Ref {
	var refs []ast.Ref

	for _, rule := range module.Rules {
		var ref ast.Ref

		for _, term := range rule.Head.Ref().GroundPrefix() {
			ref = ref.Append(term)

			name := ""

			switch v := term.Value.(type) {
			case ast.Var:
				name = string(v)
			case ast.String:
				name = string(v)
			}

			if strings.HasPrefix(name, tester.TestPrefix) || strings.HasPrefix(name, tester.SkipTestPrefix) {
				refs = append(refs, module.Package.Path.Extend(ref))

				break
			}
		}
	}

	return refs
}

// collectTestResults collects test results from the runner's channel.
func collectTestResults(ch chan *tester.Result) []tester.Result {
	results := make([]tester.Result, 0)

	for tr := range ch {
		// Clear trace (not needed in response, potentially large)
		tr.Trace = nil
		results = append(results, *tr)
	}

	return results
}

func newStoreAndTxn(ctx context.Context, cfg *config.Config) (storage.Store, storage.Transaction) {
	store := inmem.NewFromObjectWithOpts(map[string]any{
		"internal": map[string]any{
			"capabilities": outil.Or(cfg.Capabilities, config.CapabilitiesForThisVersion),
		},
	}, inmem.OptRoundTripOnWrite(false), inmem.OptReturnASTValuesOnRead(true))

	txn, _ := store.NewTransaction(ctx, storage.WriteParams)

	return store, txn
}
