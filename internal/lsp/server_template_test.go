package lsp

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/connection"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

type message struct {
	method string
	bytes  []byte
}

func TestNewFileTemplating(t *testing.T) {
	t.Parallel()

	// We have managed to get most tests passing on windows in
	// https://github.com/open-policy-agent/regal/pull/1741
	// This one is still flaking and we can come back later to address.
	if runtime.GOOS == "windows" {
		t.Skip("skipping TestNewFileTemplating on Windows")
	}

	files := map[string]string{
		".regal/config.yaml": `rules:
  idiomatic:
    directory-package-mismatch:
      level: error
      exclude-test-suffix: false
`,
	}

	tempDir := testutil.TempDirectoryOf(t, files)

	receivedMessages := make(chan message, 10)
	ls, connClient, ctx := createAndInitServer(t, tempDir, createTemplateTestClientHandler(t, receivedMessages))

	ls.StartConfigWorker(ctx)

	// TODO: In an ideal world, we would be able to start with other workers,
	// but there are some tests that test this functionality in detail and they
	// do not want this running by default.
	ls.StartTemplateWorker(ctx)

	// Wait for the custom config to load. We check that directory-package-mismatch has
	// level "error" (set in the test config) to ensure the custom config is loaded before
	// the template worker processes the file, ensuring it generates the expected rename operations.
	timeout := time.NewTimer(determineTimeout())
	select {
	case <-timeout.C:
		t.Fatal("timed out waiting for server to load config")
	default:
		for {
			time.Sleep(testPollInterval)

			cfg := ls.getLoadedConfig()
			if cfg != nil {
				if rule, ok := cfg.Rules["idiomatic"]["directory-package-mismatch"]; ok && rule.Level == "error" {
					if exclude, ok := rule.Extra["exclude-test-suffix"].(bool); ok && !exclude {
						break
					}
				}
			}
		}
	}

	// Touch the new file on disk
	newFilePath := filepath.Join(tempDir, "foo", "bar", "policy_test.rego")
	newFileURI := uri.FromPath(clients.IdentifierGeneric, newFilePath)
	expectedNewFileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(
		tempDir, "foo", "bar_test", "policy_test.rego",
	))

	must.MkdirAll(t, filepath.Dir(newFilePath))
	must.WriteFile(t, newFilePath, []byte(""))

	params := types.CreateFilesParams{Files: []types.File{{URI: newFileURI}}}

	// Client sends workspace/didCreateFiles notification
	if err := connClient.Notify(ctx, "workspace/didCreateFiles", params, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// Validate that the client received a workspace edit
	timeout.Reset(determineTimeout())

	// Construct proper URI for delete operation
	deleteTargetPath := filepath.Join(tempDir, "foo", "bar")
	deleteTargetURI := uri.FromPath(clients.IdentifierGeneric, deleteTargetPath)

	expectedMessage := fmt.Sprintf(`{
  "edit": {
    "documentChanges": [
      {
        "edits": [
          {
            "newText": "package foo.bar_test\n\n",
            "range": {
              "end": {
                "character": 0,
                "line": 0
              },
              "start": {
                "character": 0,
                "line": 0
              }
            }
          }
        ],
        "textDocument": {
          "uri": "%[1]s",
          "version": null
        }
      },
      {
        "kind": "rename",
        "newUri": "%[2]s",
        "oldUri": "%[1]s"
      },
      {
        "kind": "delete",
        "options": {
          "ignoreIfNotExists": true,
          "recursive": true
        },
        "uri": "%[3]s"
      }
    ]
  },
  "label": "Template new Rego file"
}`, newFileURI, expectedNewFileURI, deleteTargetURI)

	for success := false; !success; {
		select {
		case msg := <-receivedMessages:
			if msg.method != "workspace/applyEdit" {
				t.Logf("waiting for 'workspace/applyEdit' message, got %s, keep waiting", msg.method)

				continue
			}

			t.Log("received message:", string(msg.bytes))

			expectedLines := strings.Split(expectedMessage, "\n")
			gotLines := strings.Split(string(msg.bytes), "\n")

			if len(gotLines) != len(expectedLines) {
				t.Logf("expected message to have %d lines, got %d", len(expectedLines), len(gotLines))

				continue
			}

			allLinesMatch := true

			for i, expected := range expectedLines {
				if gotLines[i] != expected {
					t.Logf("expected message line %d to be:\n%s\ngot\n%s", i, expected, gotLines[i])

					allLinesMatch = false
				}
			}

			success = allLinesMatch
		case <-timeout.C:
			t.Log("never received expected message", expectedMessage)
			t.Fatal("timed out waiting for expected message to be sent")
		}
	}
}

// TestTemplateWorkerSkipsDidOpenWhenTemplating tests the race condition fix for
// https://github.com/open-policy-agent/regal/issues/1608 where didOpen would overwrite
// templated content in cache (with "") before the template worker could
// complete.
func TestTemplateWorkerSkipsDidOpenWhenTemplating(t *testing.T) {
	t.Parallel()

	receivedMessages := receivedMessagesMap{"policy.rego": make(chan []string, 10)}
	tempDir := testutil.TempDirectoryOf(t, map[string]string{".regal/config.yaml": `{}`})
	clientHandler := createPublishDiagnosticsHandler(t, test.DebugLogger(t), receivedMessages)
	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	// note that the template worker is not started, we simulate it's running
	// manually in this test.

	newFilePath := filepath.Join(tempDir, "foo", "bar", "policy.rego")
	newFileURI := uri.FromPath(clients.IdentifierGeneric, newFilePath)

	must.MkdirAll(t, filepath.Dir(newFilePath))
	must.WriteFile(t, newFilePath, []byte(""))

	// Wait for server initialization to complete
	timeout := time.NewTimer(determineTimeout())
	select {
	case <-timeout.C:
		t.Fatal("timed out waiting for server to initialize")
	default:
		for ls.getLoadedConfig() == nil {
			time.Sleep(testPollInterval)
		}
	}

	// Set initial cache content so linting can proceed
	initialContent := "package foo.bar\n\ninitial := true\n"
	ls.cache.SetFileContents(newFileURI, initialContent)

	// Simulate templating in progress
	ls.templater.MarkInProgress(newFileURI)

	params := types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{URI: newFileURI, LanguageID: "rego", Version: 1, Text: ""},
	}

	// Send didOpen while "templating" - should be skipped
	if err := connClient.Notify(ctx, "textDocument/didOpen", params, nil); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	// Wait for didOpen to complete
	timeout = time.NewTimer(determineTimeout())
	waitForViolations(t, "policy.rego", []string{}, []string{}, timeout, receivedMessages)

	// Drain any additional pending diagnostics messages from workspace linting
	// to avoid race condition where buffered messages from first didOpen are
	// consumed by the second waitForViolations call
	drainMessages(receivedMessages["policy.rego"])

	// Cache should still have initial content because didOpen was skipped
	cacheContent := testutil.MustBeOK(ls.cache.GetFileContents(newFileURI))(t)
	if cacheContent != initialContent {
		t.Fatalf("didOpen should have been skipped but cache changed from %q to %q", initialContent, cacheContent)
	}

	// Simulate template worker completing
	expectedTemplateContent := "package foo.bar\n\n"
	ls.cache.SetFileContents(newFileURI, expectedTemplateContent)
	ls.templater.ClearInProgress(newFileURI)

	// Verify templated content is preserved
	cacheContent = testutil.MustBeOK(ls.cache.GetFileContents(newFileURI))(t)
	if cacheContent != expectedTemplateContent {
		t.Fatalf("expected cache to contain %q, got %q", expectedTemplateContent, cacheContent)
	}

	// Now didOpen should work normally
	newContent := "package foo.bar\n\nimport rego.v1\n"
	openParams := types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{URI: newFileURI, LanguageID: "rego", Version: 2, Text: newContent},
	}

	if err := connClient.Notify(ctx, "textDocument/didOpen", openParams, nil); err != nil {
		t.Fatalf("failed to send second didOpen notification: %s", err)
	}

	// Wait for second didOpen to complete
	timeout = time.NewTimer(determineTimeout())
	waitForViolations(t, "policy.rego", []string{}, []string{}, timeout, receivedMessages)

	// Drain any pending messages before checking final state
	drainMessages(receivedMessages["policy.rego"])

	// Verify cache was updated
	finalContent := testutil.MustBeOK(ls.cache.GetFileContents(newFileURI))(t)
	if finalContent != newContent {
		t.Fatalf("expected cache to contain %q, got %q", newContent, finalContent)
	}
}

func createTemplateTestClientHandler(t *testing.T, receivedMessages chan message) connection.HandlerFunc {
	t.Helper()

	return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		receivedMessages <- message{
			method: req.Method,
			bytes:  must.Marshal(t, req.Params, jsontext.WithIndent("  ")),
		}

		return struct{}{}, nil
	}
}
