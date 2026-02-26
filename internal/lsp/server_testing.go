package lsp

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/open-policy-agent/opa/v1/runtime/info"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/lsp/types"
)

// handleRunTests handles the regal/runTests LSP request.
// It runs OPA tests based on the provided parameters and returns results.
func (l *LanguageServer) handleRunTests(
	ctx context.Context,
	params types.RunTestsParams,
) (any, error) {
	modules := l.cache.GetAllModules()

	txn, err := l.regoStore.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	defer l.regoStore.Abort(ctx, txn)

	runtimeInfo, err := info.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime info: %w", err)
	}

	filter := fmt.Sprintf("%s.%s$", regexp.QuoteMeta(params.Package), regexp.QuoteMeta(params.Name))

	runner := tester.NewRunner().
		SetCompiler(l.testingCompiler).
		SetStore(l.regoStore).
		SetModules(modules).
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
