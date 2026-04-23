package lsp

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/runtime/info"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
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
	if _, ok := l.cache.GetModule(params.URI); !ok {
		if _, err := updateParse(ctx, l.parseOpts(params.URI, l.builtinsForCurrentCapabilities())); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", params.URI, err)
		}
	}

	store, txn := newStoreAndTxn(ctx, l.getLoadedConfig())

	defer store.Abort(ctx, txn)

	filter := fmt.Sprintf("%s.%s$", regexp.QuoteMeta(params.Package), regexp.QuoteMeta(params.Name))

	runner := tester.NewRunner().
		SetCompiler(compile.NewCompilerWithRegalBuiltins().
			WithEnablePrintStatements(true).
			WithUseTypeCheckAnnotations(true)).
		SetStore(store).
		SetBundles(l.assembleBundles()).
		SetRuntime(runtimeInfo()).
		CapturePrintOutput(true).
		SetTimeout(5 * time.Second).
		Filter(filter)

	ch, err := runner.RunTests(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	return collectTestResults(ch), nil
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
			"capabilities": util.Or(cfg.Capabilities, config.CapabilitiesForThisVersion),
		},
	}, inmem.OptRoundTripOnWrite(false), inmem.OptReturnASTValuesOnRead(true))

	txn, _ := store.NewTransaction(ctx, storage.WriteParams)

	return store, txn
}
