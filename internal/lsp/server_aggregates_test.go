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
	"github.com/open-policy-agent/regal/internal/lsp/types"
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
	messages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), messages)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// no unresolved-imports at this stage
	waitForViolationStatus(t, "foo.rego", "unresolved-import", false, timeout, messages)

	notifyDocumentChange(t, connClient, filepath.Join(tempDir, "bar.rego"), "package qux")

	// unresolved-imports is now expected
	timeout.Reset(determineTimeout())
	waitForViolationStatus(t, "foo.rego", "unresolved-import", true, timeout, messages)

	notifyDocumentChange(t, connClient, filepath.Join(tempDir, "foo.rego"), `package foo

import data.baz
import data.qux # new name for bar.rego package
`)

	// unresolved-imports is again not expected
	timeout.Reset(determineTimeout())
	waitForViolationStatus(t, "foo.rego", "unresolved-import", false, timeout, messages)
}

func waitForViolationStatus(t *testing.T, key, rule string, want bool, timeout *time.Timer, messages messages) {
	t.Helper()

	for success := false; !success; {
		select {
		case violations := <-messages[key]:
			if want && !slices.Contains(violations, rule) {
				t.Log("waiting for violations to contain ", rule)

				continue
			}

			if !want && slices.Contains(violations, rule) {
				t.Log("waiting for violations to not contain ", rule)

				continue
			}

			success = true
		case <-timeout.C:
			if want {
				t.Fatalf("timed out waiting for violations to contain %s", rule)
			}

			t.Fatalf("timed out waiting for violations to not contain %s", rule)
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

	clientHandler := func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (result any, err error) {
		return struct{}{}, nil
	}

	files := map[string]string{
		"foo.rego":           "package foo\n\nimport data.baz\n",
		"bar.rego":           "package bar\n\nimport data.quz\n",
		".regal/config.yaml": "",
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
			aggregates := ls.cache.GetFileAggregates()
			if aggregates == nil || aggregates.Len() == 0 {
				t.Log("no server aggregates")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatal("timed out waiting for file aggregates to be set")
		}
	}

	aggregates := ls.cache.GetFileAggregates()

	imports := determineImports(t, aggregates)
	if exp := []string{"baz", "quz"}; !slices.Equal(exp, imports) {
		t.Fatalf("global state imports unexpected, got %v exp %v", imports, exp)
	}

	// 2. check the aggregates for a file are updated after an update
	notifyDocumentChange(t, connClient, filepath.Join(tempDir, "bar.rego"), `package bar

import data.qux # changed
import data.wow # new
`)

	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case <-ticker.C:
			aggregates := ls.cache.GetFileAggregates()

			imports = determineImports(t, aggregates)

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
	notifyDocumentChange(t, connClient, filepath.Join(tempDir, "bar.rego"), `package bax # package imported in foo.rego

import rego.v1
`)

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
	bar := filepath.Join(tempDir, "bar.rego")
	notifyDocumentChange(t, connClient, bar, `package bar # original package to bring back the violation

import rego.v1
`)

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

func toStringSlice(arr *ast.Array) []string {
	result := make([]string, 0, arr.Len())
	for i := range arr.Len() {
		if str, ok := arr.Elem(i).Value.(ast.String); ok {
			result = append(result, string(str))
		}
	}

	return result
}

func notifyDocumentChange(t *testing.T, connClient *jsonrpc2.Conn, path, newContents string) {
	t.Helper()

	err := connClient.Notify(t.Context(), "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument:   types.VersionedTextDocumentIdentifier{URI: uri.FromPath(clients.IdentifierGoTest, path)},
		ContentChanges: []types.TextDocumentContentChangeEvent{{Text: newContents}},
	}, nil)
	if err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}
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
