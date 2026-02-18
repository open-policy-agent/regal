package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"

	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
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
// This is called AFTER the parse has completed in the lintFileJobs worker.
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
		// This shouldn't happen since parse completed successfully before we were called,
		// but handle it just in case
		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	// Prepare the AST input for the query
	//nolint:staticcheck
	inputMap, err := rparse.PrepareAST(l.toRelativePath(fileURI), contents, module)
	if err != nil {
		l.log.Message("failed to prepare AST for test locations: %s", err)

		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	// Get or prepare the test locations query
	pq, err := l.queryCache.GetOrSet(ctx, l.regoStore, query.TestLocations)
	if err != nil {
		return fmt.Errorf("failed to prepare query %s: %w", query.TestLocations, err)
	}

	// Execute the query
	resultSet, err := pq.EvalQuery().Eval(ctx, rego.EvalInput(inputMap))
	if err != nil {
		l.log.Message("failed to evaluate test locations query: %s", err)

		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	if len(resultSet) == 0 || len(resultSet[0].Expressions) == 0 {
		return l.sendTestLocations(ctx, fileURI, []any{})
	}

	// Send the raw result value
	result := resultSet[0].Expressions[0].Value

	return l.sendTestLocations(ctx, fileURI, result)
}

func (l *LanguageServer) sendTestLocations(ctx context.Context, fileURI string, locations any) error {
	params := map[string]any{
		"uri":       fileURI,
		"locations": locations,
	}

	// Extract line numbers for logging
	var lines []int
	if locSlice, ok := locations.([]any); ok {
		for _, loc := range locSlice {
			if locMap, ok := loc.(map[string]any); ok {
				if location, ok := locMap["location"].(map[string]any); ok {
					if row, ok := location["row"].(int); ok {
						lines = append(lines, row)
					} else if rowFloat, ok := location["row"].(float64); ok {
						lines = append(lines, int(rowFloat))
					} else if rowNum, ok := location["row"].(json.Number); ok {
						if rowInt, err := rowNum.Int64(); err == nil {
							lines = append(lines, int(rowInt))
						}
					}
				}
			}
		}
	}

	if len(lines) > 0 {
		l.log.Message("sending test locations notification for %s (lines: %v)", fileURI, lines)
	} else {
		l.log.Message("sending test locations notification for %s (no tests found)", fileURI)
	}

	if err := l.conn.Notify(ctx, "regal/testLocations", params); err != nil {
		return fmt.Errorf("failed to send test locations notification: %w", err)
	}

	return nil
}
