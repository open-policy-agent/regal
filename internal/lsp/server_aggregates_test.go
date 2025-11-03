package lsp

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/report"
)

func TestLanguageServerLintsUsingAggregateState(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego": `package foo

import rego.v1

import data.bar
import data.baz
`,
		"bar.rego": `package bar

import rego.v1
`,
		"baz.rego": `package baz

import rego.v1
`,
		".regal/config.yaml": ``,
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	messages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), messages)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// no unresolved-imports at this stage
	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if slices.Contains(violations, "unresolved-import") {
				t.Logf("waiting for violations to not contain unresolved-import")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for expected foo.rego diagnostics")
		}
	}

	barURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego"))

	if err := connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: barURI,
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package qux

import rego.v1
`,
			},
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// unresolved-imports is now expected
	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if !slices.Contains(violations, "unresolved-import") {
				t.Log("waiting for violations to contain unresolved-import")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for expected foo.rego diagnostics")
		}
	}

	fooURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "foo.rego"))

	if err := connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: fooURI,
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package foo

import rego.v1

import data.baz
import data.qux # new name for bar.rego package
`,
			},
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// unresolved-imports is again not expected
	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if slices.Contains(violations, "unresolved-import") {
				t.Log("waiting for violations to not contain unresolved-import")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for expected foo.rego diagnostics")
		}
	}
}

// Test to ensure that annotations are parsed correctly.
func TestRulesWithMetadataNotReportedForMissingMeta(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego": `# METADATA
# title: foo
package foo
`,
		"bar.rego": `# METADATA
# title: foo
package bar
`,
		".regal/config.yaml": `rules:
  custom:
    missing-metadata:
      level: error
  idiomatic:
    directory-package-mismatch:
      level: ignore
`,
	}

	messages := createMessageChannels(files)
	logger := log.NewLogger(log.LevelDebug, t.Output())
	clientHandler := createPublishDiagnosticsHandler(t, logger, messages)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tempDir := testutil.TempDirectoryOf(t, files)

	createAndInitServer(t, ctx, tempDir, clientHandler)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// no missing-metadata
	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if len(violations) > 0 {
				t.Logf("unexpected violations for foo.rego: %v, waiting...", violations)
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for expected foo.rego diagnostics")
		}
	}
}

func TestLanguageServerUpdatesAggregateState(t *testing.T) {
	t.Parallel()

	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		t.Logf("message %s", req.Method)

		return struct{}{}, nil
	}

	files := map[string]string{
		"foo.rego": `package foo

import rego.v1

import data.baz
`,
		"bar.rego": `package bar

import rego.v1

import data.quz
`,
		".regal/config.yaml": ``,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tempDir := testutil.TempDirectoryOf(t, files)
	ls, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	// 1. check the Aggregates are set at start up
	timeout := time.NewTimer(determineTimeout())

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for success := false; !success; {
		select {
		case <-ticker.C:
			aggs := ls.cache.GetFileAggregates()
			if len(aggs) == 0 {
				t.Logf("server aggregates %d", len(aggs))

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for file aggregates to be set")
		}
	}

	determineImports := func(aggs map[string][]report.Aggregate) []string {
		imports := []string{}

		unresolvedImportAggs, ok := aggs["imports/unresolved-import"]
		if !ok {
			t.Fatalf("expected imports/unresolved-import aggregate data")
		}

		for _, entry := range unresolvedImportAggs {
			if aggregateData, ok := entry["aggregate_data"].(map[string]any); ok {
				if importsList, ok := aggregateData["imports"].([]any); ok {
					for _, imp := range importsList {
						item := testutil.MustBe[[]any](t, imp)
						if pathList, ok := item[0].([]any); ok {
							pathParts := []string{}

							for _, p := range pathList {
								if pathStr, ok := p.(string); ok {
									pathParts = append(pathParts, pathStr)
								}
							}

							imports = append(imports, strings.Join(pathParts, "."))
						}
					}
				}
			}
		}

		return util.Sorted(imports)
	}

	imports := determineImports(ls.cache.GetFileAggregates())
	if exp := []string{"baz", "quz"}; !slices.Equal(exp, imports) {
		t.Fatalf("global state imports unexpected, got %v exp %v", imports, exp)
	}

	// 2. check the aggregates for a file are updated after an update
	if err := connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego")),
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package bar

import rego.v1

import data.qux # changed
import data.wow # new
`,
			},
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case <-ticker.C:
			imports = determineImports(ls.cache.GetFileAggregates())

			if exp, got := []string{"baz", "qux", "wow"}, imports; !slices.Equal(exp, got) {
				t.Logf("global state imports unexpected, got %v exp %v", got, exp)

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for file aggregates to be set")
		}
	}
}

func TestLanguageServerAggregateViolationFixedAndReintroducedInUnviolatingFileChange(t *testing.T) {
	t.Parallel()

	var err error

	files := map[string]string{
		"foo.rego": `package foo

import rego.v1

import data.bax # initially unresolved-import

variable = "string" # use-assignment-operator
`,
		"bar.rego": `package bar

import rego.v1
`,
		".regal/config.yaml": ``,
	}

	logger := log.NewLogger(log.LevelDebug, t.Output())
	tempDir := testutil.TempDirectoryOf(t, files)
	messages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, logger, messages)

	// set up the server and client connections
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	// wait for foo.rego to have the correct violations
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	for success := true; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if !slices.Contains(violations, "unresolved-import") {
				t.Logf("waiting for violations to contain unresolved-import")

				continue
			}

			if !slices.Contains(violations, "use-assignment-operator") {
				t.Logf("waiting for violations to contain use-assignment-operator")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for foo.rego diagnostics")
		}
	}

	// update the contents of the bar.rego file to address the unresolved-import
	barURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego"))

	err = connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: barURI,
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package bax # package imported in foo.rego

import rego.v1
`,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// wait for foo.rego to have the correct violations
	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if slices.Contains(violations, "unresolved-import") {
				t.Logf("waiting for violations to not contain unresolved-import")

				continue
			}

			if !slices.Contains(violations, "use-assignment-operator") {
				t.Logf("use-assignment-operator should still be present")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for foo.rego diagnostics")
		}
	}

	// update the contents of the bar.rego to bring back the violation
	err = connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.TextDocumentIdentifier{
			URI: barURI,
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package bar # original package to bring back the violation

import rego.v1
`,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// check the violation is back
	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if !slices.Contains(violations, "unresolved-import") {
				t.Logf("waiting for violations to contain unresolved-import")

				continue
			}

			if !slices.Contains(violations, "use-assignment-operator") {
				t.Logf("use-assignment-operator should still be present")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for foo.rego diagnostics")
		}
	}
}
