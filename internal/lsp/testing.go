package lsp

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	rparse "github.com/open-policy-agent/regal/internal/parse"
)

func (l *LanguageServer) StartTestLocationsWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.testLocationJobs:
			if err := l.processTestLocationsUpdate(ctx, job.URI); err != nil {
				l.log.Message("failed to process test locations: %s", err)
			}
		}
	}
}

// processTestLocationsUpdate queries for test rule locations and sends them to the client.
// This is called after the parse has completed in the lintFileJobs worker to
// ensure we send the latest.
func (l *LanguageServer) processTestLocationsUpdate(ctx context.Context, fileURI string) error {
	if l.ignoreURI(fileURI) {
		return nil
	}

	if ok := l.cache.HasFileContents(fileURI); !ok {
		return nil
	}

	contents, ok := l.cache.GetFileContents(fileURI)
	if !ok {
		return fmt.Errorf("failed to get file contents for uri %q", fileURI)
	}

	module, ok := l.cache.GetModule(fileURI)
	if !ok {
		// This shouldn't happen since parsing completed successfully before
		// calling
		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	// TODO: avoid calling this as it's deprecated
	//nolint:staticcheck
	inputMap, err := rparse.PrepareAST(l.toRelativePath(fileURI), contents, module)
	if err != nil {
		l.log.Message("failed to prepare AST for test locations: %s", err)

		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	pq, err := l.queryCache.GetOrSet(ctx, l.regoStore, query.TestLocations)
	if err != nil {
		return fmt.Errorf("failed to prepare query %s: %w", query.TestLocations, err)
	}

	resultSet, err := pq.EvalQuery().Eval(ctx, rego.EvalInput(inputMap))
	if err != nil {
		l.log.Message("failed to evaluate test locations query: %s", err)

		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	if len(resultSet) == 0 || len(resultSet[0].Expressions) == 0 {
		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	result := resultSet[0].Expressions[0].Value

	return l.sendTestLocations(ctx, fileURI, result)
}

func (l *LanguageServer) sendTestLocations(ctx context.Context, fileURI string, locations any) error {
	params := map[string]any{
		"uri":       fileURI,
		"locations": locations,
	}

	if err := l.conn.Notify(ctx, "regal/testLocations", params); err != nil {
		return fmt.Errorf("failed to send test locations notification: %w", err)
	}

	return nil
}
