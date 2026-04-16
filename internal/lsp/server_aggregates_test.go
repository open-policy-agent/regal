package lsp

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func TestLanguageServerLintsUsingAggregateState(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego":           "package foo\nimport data.bar\nimport data.baz",
		"bar.rego":           "package bar",
		"baz.rego":           "package baz",
		".regal/config.yaml": "",
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), receivedMessages)

	_, connClient, _ := createAndInitServer(t, tempDir, clientHandler)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// no unresolved-imports at this stage
	waitForViolations(t, "foo.rego", []string{}, []string{"unresolved-import"}, timeout, receivedMessages)

	notifyDocumentChange(
		t,
		connClient,
		uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego")),
		"package qux",
	)

	// unresolved-imports is now expected
	timeout.Reset(determineTimeout())
	waitForViolations(t, "foo.rego", []string{"unresolved-import"}, []string{}, timeout, receivedMessages)

	notifyDocumentChange(
		t,
		connClient,
		uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "foo.rego")),
		`package foo

import data.baz
import data.qux # new name for bar.rego package
`)

	// unresolved-imports is again not expected
	timeout.Reset(determineTimeout())
	waitForViolations(t, "foo.rego", []string{}, []string{"unresolved-import"}, timeout, receivedMessages)
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

	receivedMessages := createMessageChannels(files)
	logger := log.NewLogger(log.LevelDebug, t.Output())
	clientHandler := createPublishDiagnosticsHandler(t, logger, receivedMessages)

	tempDir := testutil.TempDirectoryOf(t, files)

	ls, _, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// Wait for custom config to load with directory-package-mismatch set to ignore
	// and missing-metadata set to error
	select {
	case <-timeout.C:
		t.Fatalf("timed out waiting for server to load config")
	default:
		for {
			time.Sleep(testPollInterval)

			cfg := ls.getLoadedConfig()
			if cfg != nil {
				// Verify directory-package-mismatch is ignored
				if rule, ok := cfg.Rules["idiomatic"]["directory-package-mismatch"]; ok && rule.Level == "ignore" {
					// Also verify missing-metadata is set to error
					if mmRule, ok := cfg.Rules["custom"]["missing-metadata"]; ok && mmRule.Level == "error" {
						break
					}
				}
			}
		}
	}

	timeout.Reset(determineTimeout())

	// no missing-metadata
	waitForViolations(t, "foo.rego", []string{}, []string{}, timeout, receivedMessages)
}

func TestLanguageServerUpdatesAggregateState(t *testing.T) {
	t.Parallel()

	clientHandler := func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (result any, err error) {
		return struct{}{}, nil
	}

	files := map[string]string{
		"foo.rego":           "package foo\n\nimport data.baz\n",
		"bar.rego":           "package bar\n\nimport data.quz\n",
		".regal/config.yaml": "",
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	// 1. check the Aggregates are set at start up
	// should be here now since only when workspace lint done, does the above return
	aggs, err := GetAST[ast.Object](ctx, ls.regoStore, pathWorkspaceAggregates)
	if err != nil {
		t.Fatalf("failed to get file aggregates: %v", err)
	}

	if aggs == nil {
		t.Fatalf("expected aggregates to be set")
	}

	imports := determineImports(t, aggs)
	if exp := []string{"baz", "quz"}; !slices.Equal(exp, imports) {
		t.Fatalf("global state imports unexpected, got %v exp %v", imports, exp)
	}

	// 2. check the aggregates for a file are updated after an update
	notifyDocumentChange(
		t,
		connClient,
		uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego")),
		`package bar

import data.qux # changed
import data.wow # new
`)

	timeout := time.NewTimer(determineTimeout())

	ticker := time.NewTicker(testPollInterval)
	defer ticker.Stop()

	pollCount := 0
	for success := false; !success; {
		select {
		case <-ticker.C:
			pollCount++
			aggs, err := GetAST[ast.Object](ctx, ls.regoStore, pathWorkspaceAggregates)
			if err != nil {
				t.Fatalf("failed to get file aggregates: %v", err)
			}

			if aggs == nil {
				t.Logf("aggregates not set yet (poll %d)", pollCount)

				continue
			}

			imports = determineImports(t, aggs)

			if exp, got := []string{"baz", "qux", "wow"}, imports; !slices.Equal(exp, got) {
				t.Logf("global state imports unexpected, got %v exp %v (poll %d)", got, exp, pollCount)

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for file aggregates to be set after %d polls", pollCount)
		}
	}
}

func TestLanguageServerAggregateViolationFixedAndReintroducedInUnviolatingFileChange(t *testing.T) {
	t.Parallel()

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
	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, logger, receivedMessages)

	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	// wait for foo.rego to have the correct violations
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	waitForViolations(
		t,
		"foo.rego",
		[]string{"unresolved-import", "use-assignment-operator"},
		[]string{},
		timeout,
		receivedMessages,
	)

	// update the contents of the bar.rego file to address the unresolved-import
	notifyDocumentChange(
		t,
		connClient,
		uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego")),
		`package bax # package imported in foo.rego

import rego.v1
`)

	// wait for foo.rego to have the correct violations
	timeout.Reset(determineTimeout())

	waitForViolations(
		t,
		"foo.rego",
		[]string{"use-assignment-operator"},
		[]string{"unresolved-import"},
		timeout,
		receivedMessages,
	)

	// update the contents of the bar.rego to bring back the violation
	barURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego"))
	notifyDocumentChange(t, connClient, barURI, `package bar # original package to bring back the violation

import rego.v1
`)

	// check the violation is back
	timeout.Reset(determineTimeout())

	waitForViolations(
		t,
		"foo.rego",
		[]string{"unresolved-import", "use-assignment-operator"},
		[]string{},
		timeout,
		receivedMessages,
	)
}

func toStringSlice(arr *ast.Array) []string {
	result := make([]string, 0, arr.Len())
	for i := range arr.Len() {
		if str, ok := arr.Elem(i).Value.(ast.String); ok {
			result = append(result, string(str))
		}
	}

	return result
}

func determineImports(t *testing.T, aggs ast.Object) (imports []string) {
	t.Helper()

	for _, fileURI := range aggs.Keys() {
		fileAggs := aggs.Get(fileURI).Value.(ast.Object)

		if aggregates, ok := rast.GetValue[ast.Set](fileAggs, "imports/unresolved-import"); ok {
			for aggregate := range rast.ValuesOfType[ast.Object](aggregates.Slice()) {
				if importsList, ok := rast.GetValue[*ast.Array](aggregate, "imports"); ok {
					importsList.Foreach(func(entry *ast.Term) {
						if arrEntry, ok := entry.Value.(*ast.Array); ok && arrEntry.Len() > 0 {
							if pathArr, ok := arrEntry.Elem(0).Value.(*ast.Array); ok {
								imports = append(imports, strings.Join(toStringSlice(pathArr), "."))
							}
						}
					})
				}
			}
		}
	}

	return util.Sorted(imports)
}
