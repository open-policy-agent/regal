package lsp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// TestLanguageServerSingleFile tests that changes to a single file and Regal config are handled correctly by the
// language server by making updates to both and validating that the correct diagnostics are sent to the client.
//
// This test also ensures that updating the config to point to a non-default engine and capabilities version works
// and causes that engine's builtins to work with completions.
//

func TestLanguageServerSingleFile(t *testing.T) {
	t.Parallel()

	mainRegoContents := `package main

import rego.v1
allow = true
`

	files := map[string]string{
		"main.rego": mainRegoContents,
		".regal/config.yaml": `
rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore`,
	}

	// set up the workspace content with some example rego and regal config
	tempDir := testutil.TempDirectoryOf(t, files)
	mainRegoURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tempDir, filepath.FromSlash(mainRegoFileName)))

	receivedMessages := make(chan types.FileDiagnostics, defaultBufferedChannelSize)
	clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedMessages))

	_, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	// validate that the client received a diagnostics notification for the file
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	waitForDiagnostics(t, receivedMessages, mainRegoURI, []string{"opa-fmt", "use-assignment-operator"}, timeout)

	// Client sends textDocument/didChange notification with new contents for main.rego
	// no response to the call is expected
	notifyDocumentChange(t, connClient, mainRegoURI, `package main
import rego.v1
allow := true
`)

	// validate that the client received a new diagnostics notification for the file
	timeout.Reset(determineTimeout())

	waitForDiagnostics(t, receivedMessages, mainRegoURI, []string{"opa-fmt"}, timeout)

	// config update is caught by the config watcher
	newConfigContents := `
rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  style:
    opa-fmt:
      level: ignore
`

	must.WriteFile(t, filepath.Join(tempDir, ".regal", "config.yaml"), []byte(newConfigContents))

	// validate that the client received a new, empty diagnostics notification for the file
	timeout.Reset(determineTimeout())

	waitForDiagnostics(t, receivedMessages, mainRegoURI, []string{}, timeout)

	// Client sends new config with an EOPA capabilities file specified.
	newConfigContents = `
rules:
  style:
    opa-fmt:
      level: ignore
  idiomatic:
    directory-package-mismatch:
      level: ignore
capabilities:
  from:
    engine: eopa
    version: v1.23.0
`

	must.WriteFile(t, filepath.Join(tempDir, ".regal", "config.yaml"), []byte(newConfigContents))

	// validate that the client received a new, empty diagnostics notification for the file
	timeout.Reset(determineTimeout())

	waitForDiagnostics(t, receivedMessages, mainRegoURI, []string{}, timeout)

	// Client sends textDocument/didChange notification with new
	// contents for main.rego no response to the call is expected. We added
	// the start of an EOPA-specific call, so if the capabilities were
	// loaded correctly, we should see a completion later after we ask for
	// it.
	notifyDocumentChange(t, connClient, mainRegoURI, `package main
import rego.v1

# METADATA
# entrypoint: true
allow := neo4j.q
`)

	// validate that the client received a new diagnostics notification for the file
	timeout.Reset(determineTimeout())

	waitForDiagnostics(t, receivedMessages, mainRegoURI, []string{}, timeout)

	// 7. With our new config applied, and the file updated, we can ask the
	// LSP for a completion. We expect to see neo4j.query show up. Since
	// neo4j.query is an EOPA-specific builtin, it should never appear if
	// we're using the normal OPA capabilities file.
	timeout.Reset(determineTimeout())

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for success := false; !success; {
		select {
		case <-ticker.C:
			// Create a new context with timeout for each request, this is
			// timed out after using the default as the GHA runner is super
			// slow in the race detector
			reqCtx, reqCtxCancel := context.WithTimeout(ctx, determineTimeout())

			resp := make(map[string]any)
			params := types.NewCompletionParams(mainRegoURI, 5, 16, nil)
			err := connClient.Call(reqCtx, "textDocument/completion", params, &resp)

			reqCtxCancel()
			must.Equal(t, nil, err, "failed to send completion request: %s", err)

			itemsList := must.Be[[]any](t, resp["items"])

			for _, itemI := range itemsList {
				item := must.Be[map[string]any](t, itemI)
				label := must.Be[string](t, item["label"])

				if label == "neo4j.query" {
					success = true

					break
				}
			}

			t.Logf("waiting for neo4j.query in completion results for neo4j.q, got %v", itemsList)
		case <-timeout.C:
			t.Fatalf("timed out waiting for file completion to correct")
		}
	}
}
