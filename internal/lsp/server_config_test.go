package lsp

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// TestLanguageServerParentDirConfig tests that regal config is loaded as it is for the
// Regal CLI, and that config files in a parent directory are loaded correctly
// even when the workspace is a child directory.
func TestLanguageServerParentDirConfig(t *testing.T) {
	t.Parallel()

	mainRegoContents := `package main

import data.test
allow := true
`

	childDirName := "child"

	files := map[string]string{
		childDirName + mainRegoFileName: mainRegoContents,
		".regal/config.yaml": `rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  style:
    opa-fmt:
      level: error
`,
	}

	// childDir will be the directory that the client is using as its workspace
	tempDir := testutil.TempDirectoryOf(t, files)
	childDir := filepath.Join(tempDir, childDirName)

	// mainRegoFileURI is used throughout the test to refer to the main.rego file
	// and so it is defined here for convenience
	mainRegoFileURI := uri.FromPath(
		clients.IdentifierGoTest,
		filepath.Join(childDir, filepath.FromSlash(mainRegoFileName)),
	)

	// set up the server and client connections
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	receivedMessages := make(chan types.FileDiagnostics, defaultBufferedChannelSize)
	clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedMessages))

	ls, _ := createAndInitServer(t, ctx, tempDir, clientHandler)

	if got, exp := ls.workspaceRootURI, uri.FromPath(ls.client.Identifier, tempDir); exp != got {
		t.Fatalf("expected client root URI to be %s, got %s", exp, got)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	for success := false; !success; {
		select {
		case requestData := <-receivedMessages:
			success = testRequestDataCodes(t, requestData, mainRegoFileURI, []string{"opa-fmt"})
		case <-timeout.C:
			t.Fatalf("timed out waiting for file diagnostics to be sent")
		}
	}

	// User updates config file contents in parent directory that is not
	// part of the workspace
	newConfigContents := `rules:
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

	for success := false; !success; {
		select {
		case requestData := <-receivedMessages:
			success = testRequestDataCodes(t, requestData, mainRegoFileURI, []string{})
		case <-timeout.C:
			t.Fatalf("timed out waiting for file diagnostics to be sent")
		}
	}
}

func TestLanguageServerCachesEnabledRulesAndUsesDefaultConfig(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDirectoryOf(t, map[string]string{
		".regal/config.yaml": `
rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  imports:
    unresolved-import:
      level: ignore
`,
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// no op handler
	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		t.Logf("message received: %s", req.Method)

		return struct{}{}, nil
	}

	ls, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	if got, exp := ls.workspaceRootURI, uri.FromPath(ls.client.Identifier, tempDir); exp != got {
		t.Fatalf("expected client root URI to be %s, got %s", exp, got)
	}

	timeout := time.NewTimer(3 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)

	for success := false; !success; {
		select {
		case <-ticker.C:
			enabledRules := ls.getEnabledNonAggregateRules()
			enabledAggRules := ls.getEnabledAggregateRules()

			if len(enabledRules) == 0 || len(enabledAggRules) == 0 {
				t.Log("no enabled rules yet...")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for enabled rules to be correct")
		}
	}

	// this event is sent to allow the server to detect the new config
	if err := connClient.Notify(ctx, "workspace/didChangeWatchedFiles", types.WorkspaceDidChangeWatchedFilesParams{
		Changes: []types.FileEvent{{
			URI:  uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, ".regal", "config.yaml")),
			Type: 1, // created
		}},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case <-ticker.C:
			enabledRules := ls.getEnabledNonAggregateRules()
			enabledAggRules := ls.getEnabledAggregateRules()

			if slices.Contains(enabledRules, "directory-package-mismatch") {
				t.Log("enabledRules still contains directory-package-mismatch")

				continue
			}

			if slices.Contains(enabledAggRules, "unresolved-import") {
				t.Log("enabledAggRules still contains unresolved-import")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for enabled rules to be correct")
		}
	}

	configContents2 := `
rules:
  style:
    opa-fmt:
      level: ignore
  idiomatic:
    directory-package-mismatch:
      level: error
  imports:
    unresolved-import:
      level: error
`

	must.WriteFile(t, filepath.Join(tempDir, ".regal", "config.yaml"), []byte(configContents2))
	timeout.Reset(determineTimeout())

	for success := false; !success; {
		select {
		case <-ticker.C:
			enabledRules := ls.getEnabledNonAggregateRules()
			enabledAggRules := ls.getEnabledAggregateRules()

			if slices.Contains(enabledRules, "opa-fmt") {
				t.Log("enabledRules still contains opa-fmt")

				continue
			}

			if !slices.Contains(enabledRules, "directory-package-mismatch") {
				t.Log("enabledRules must contain directory-package-mismatch")

				continue
			}

			if !slices.Contains(enabledAggRules, "unresolved-import") {
				t.Log("enabledAggRules must contain unresolved-import")

				continue
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for enabled rules to be correct")
		}
	}
}
