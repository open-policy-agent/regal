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
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestLanguageServerNoWorkspace(t *testing.T) {
	t.Parallel()

	mainRegoContents := `package main

import  rego.v1
allow =  true
`

	files := map[string]string{
		"foo/main.rego": mainRegoContents,
		// here we are ignoring two issues with the file, so we expect not to
		// see these but to see another issue (opa-fmt)
		".regal/config.yaml": `
rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  style:
    use-assignment-operator:
      level: ignore
`,
	}

	// set up the workspace content with some example rego and regal config
	tempDir := testutil.TempDirectoryOf(t, files)
	mainRegoURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tempDir, filepath.FromSlash(mainRegoFileName)))

	receivedMessages := make(chan types.FileDiagnostics, defaultBufferedChannelSize)
	clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedMessages))

	// set up the server and client connections
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// note using a blank tempDir here so we can simulate the single file mode
	_, connClient := createAndInitServer(t, ctx, "", clientHandler)

	// client sends textDocument/didOpen notification with contents for main.rego
	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:  mainRegoURI,
			Text: mainRegoContents,
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// validate that the client received a diagnostics notification for the file
	// with the correct items based on the settings.
	for success := false; !success; {
		select {
		case requestData := <-receivedMessages:
			success = testRequestDataCodes(t, requestData, mainRegoURI, []string{"opa-fmt"})
		case <-timeout.C:
			t.Fatalf("timed out waiting for file diagnostics to be sent")
		}
	}

	timeout.Reset(determineTimeout())
}
