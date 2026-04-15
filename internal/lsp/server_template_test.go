package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

type message struct {
	method string
	bytes  []byte
}

func TestTemplateContentsForFile(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		FileKey               string
		CacheFileContents     string
		DiskContents          map[string]string
		RequireConfig         bool
		ServerAllRegoVersions *concurrent.Map[string, ast.RegoVersion]
		ExpectedContents      string
		ExpectedError         string
	}{
		"existing contents in file": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "package foo",
			ExpectedError:     "file already has contents",
		},
		"existing contents on disk": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar.rego": "package foo",
			},
			ExpectedError: "file on disk already has contents",
		},
		"empty file is templated based on root": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar.rego":       "",
				".regal/config.yaml": "",
			},
			ExpectedContents: "package foo\n\n",
		},
		"empty test file is templated based on root": {
			FileKey:           "foo/bar_test.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar_test.rego":  "",
				".regal/config.yaml": "",
			},
			RequireConfig:    true,
			ExpectedContents: "package foo_test\n\n",
		},
		"empty deeply nested file is templated based on root": {
			FileKey:           "foo/bar/baz/bax.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar/baz/bax.rego": "",
				".regal/config.yaml":   "",
			},
			ExpectedContents: "package foo.bar.baz\n\n",
		},
		"v0 templating using rego version setting": {
			FileKey:           filepath.FromSlash("foo/bar/baz/bax.rego"),
			CacheFileContents: "",
			ServerAllRegoVersions: concurrent.MapOf(map[string]ast.RegoVersion{
				"foo": ast.RegoV0,
			}),
			DiskContents: map[string]string{
				filepath.FromSlash("foo/bar/baz/bax.rego"): "",
				filepath.FromSlash(".regal/config.yaml"):   "", // we manually set the versions, config not loaded in these tests
			},
			ExpectedContents: "package foo.bar.baz\n\nimport rego.v1\n",
		},
		"v1 templating using rego version setting": {
			FileKey:           filepath.FromSlash("foo/bar/baz/bax.rego"),
			CacheFileContents: "",
			ServerAllRegoVersions: concurrent.MapOf(map[string]ast.RegoVersion{
				"foo": ast.RegoV1,
			}),
			DiskContents: map[string]string{
				filepath.FromSlash("foo/bar/baz/bax.rego"): "",
				filepath.FromSlash(".regal/config.yaml"):   "", // we manually set the versions, config not loaded in these tests
			},
			ExpectedContents: "package foo.bar.baz\n\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			lg := log.NewLogger(log.LevelDebug, t.Output())
			td := testutil.TempDirectoryOf(t, tc.DiskContents)
			ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: lg})

			ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, td)
			ls.loadedConfigAllRegoVersions = tc.ServerAllRegoVersions

			fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, tc.FileKey))

			ls.cache.SetFileContents(fileURI, tc.CacheFileContents)

			newContents, err := ls.templateContentsForFile(fileURI)
			if tc.ExpectedError != "" {
				testutil.ErrMustContain(err, tc.ExpectedError)(t)
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if newContents != tc.ExpectedContents {
				t.Fatalf("expected contents to be\n%s\ngot\n%s", tc.ExpectedContents, newContents)
			}
		})
	}
}

func TestTemplateContentsForFileInWorkspaceRoot(t *testing.T) {
	t.Parallel()

	td := testutil.TempDirectoryOf(t, map[string]string{".regal/config.yaml": ""})
	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, td)

	fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, "foo.rego"))

	ls.cache.SetFileContents(fileURI, "")

	_, err := ls.templateContentsForFile(fileURI)

	testutil.ErrMustContain(err, "this function does not template files in the workspace root")(t)
}

func TestTemplateContentsForFileWithUnknownRoot(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})
	fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, "foo", "bar.rego"))

	ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, td)
	ls.cache.SetFileContents(fileURI, "")

	must.MkdirAll(t, td, "foo")
	assert.Equal(t, "package foo\n\n", must.Return(ls.templateContentsForFile(fileURI))(t))
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
		t.Fatalf("timed out waiting for server to load config")
	default:
		for {
			time.Sleep(100 * time.Millisecond)

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

	// Client sends workspace/didCreateFiles notification
	if err := connClient.Notify(ctx, "workspace/didCreateFiles", types.CreateFilesParams{
		Files: []types.File{{URI: newFileURI}},
	}, nil); err != nil {
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
        "oldUri": "%[1]s",
        "options": {
          "ignoreIfExists": false,
          "overwrite": false
        }
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
			t.Fatalf("timed out waiting for expected message to be sent")
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
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), receivedMessages)
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
		t.Fatalf("timed out waiting for server to initialize")
	default:
		for {
			time.Sleep(100 * time.Millisecond)

			if ls.getLoadedConfig() != nil {
				break
			}
		}
	}

	// Set initial cache content so linting can proceed
	initialContent := "package foo.bar\n\ninitial := true\n"
	ls.cache.SetFileContents(newFileURI, initialContent)

	// Simulate templating in progress
	ls.templatingFiles.Set(newFileURI, true)

	// Send didOpen while "templating" - should be skipped
	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:        newFileURI,
			LanguageID: "rego",
			Version:    1,
			Text:       "",
		},
	}, nil); err != nil {
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
	ls.templatingFiles.Delete(newFileURI)

	// Verify templated content is preserved
	cacheContent = testutil.MustBeOK(ls.cache.GetFileContents(newFileURI))(t)
	if cacheContent != expectedTemplateContent {
		t.Fatalf("expected cache to contain %q, got %q", expectedTemplateContent, cacheContent)
	}

	// Now didOpen should work normally
	newContent := "package foo.bar\n\nimport rego.v1\n"
	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:        newFileURI,
			LanguageID: "rego",
			Version:    2,
			Text:       newContent,
		},
	}, nil); err != nil {
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

func createTemplateTestClientHandler(
	t *testing.T,
	receivedMessages chan message,
) func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	t.Helper()

	return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		bs, err := json.MarshalIndent(req.Params, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal params: %s", err)
		}

		receivedMessages <- message{
			method: req.Method,
			bytes:  bs,
		}

		return struct{}{}, nil
	}
}
