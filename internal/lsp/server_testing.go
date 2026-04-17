package lsp

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/open-policy-agent/opa/v1/runtime/info"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/pkg/config"
)

// handleRunTests handles the regal/runTests LSP request.
// It runs OPA tests based on the provided parameters and returns results.
func (l *LanguageServer) handleRunTests(
	ctx context.Context,
	params types.RunTestsParams,
) (any, error) {
	// Ensure the target file is parsed before running tests.
	// This handles the case where regal/runTests is called immediately after
	// textDocument/didOpen, before the async diagnostics worker has parsed the file.
	// This makes the code more robust, but is mostly only helpful in tests.
	if _, ok := l.cache.GetModule(params.URI); !ok {
		if _, err := updateParse(ctx, l.parseOpts(params.URI, l.builtinsForCurrentCapabilities())); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", params.URI, err)
		}
	}

	// Create new isolated store for this test run (NO SHARED STATE)
	testStore := inmem.NewWithOpts(
		inmem.OptRoundTripOnWrite(false),
		inmem.OptReturnASTValuesOnRead(true),
	)

	txn, err := testStore.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	defer testStore.Abort(ctx, txn)

	// TODO: make the loading of config and regal built-ins only happen
	// in Regal repo
	cfg := l.getLoadedConfig()

	var caps *config.Capabilities
	if cfg != nil && cfg.Capabilities != nil {
		caps = cfg.Capabilities
	} else {
		caps = config.CapabilitiesForThisVersion()
	}

	_ = testStore.Write(ctx, txn, storage.AddOp, storage.MustParsePath("/internal"), map[string]any{})

	if err := testStore.Write(
		ctx,
		txn,
		storage.AddOp,
		storage.MustParsePath("/internal/capabilities"),
		caps,
	); err != nil {
		return nil, fmt.Errorf("failed to write capabilities: %w", err)
	}

	runtimeInfo, err := info.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime info: %w", err)
	}

	filter := fmt.Sprintf("%s.%s$", regexp.QuoteMeta(params.Package), regexp.QuoteMeta(params.Name))

	compiler := compile.NewCompilerWithRegalBuiltins().
		WithEnablePrintStatements(true).
		WithUseTypeCheckAnnotations(true)

	runner := tester.NewRunner().
		SetCompiler(compiler).
		SetStore(testStore).
		SetBundles(l.assembleBundles()).
		SetRuntime(runtimeInfo).
		CapturePrintOutput(true).
		SetTimeout(5 * time.Second).
		Filter(filter)

	ch, err := runner.RunTests(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	results := collectTestResults(ch)

	return results, nil
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
