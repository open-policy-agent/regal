package lsp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"main/main.rego": "package main\n\n",
	}

	logger := log.NewLogger(log.LevelDebug, t.Output())
	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, logger, receivedMessages)
	tempDir := testutil.TempDirectoryOf(t, files)
	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	mainRegoURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "main", "main.rego"))

	if err := connClient.Notify(ctx, "workspace/didChangeWatchedFiles", types.WorkspaceDidChangeWatchedFilesParams{
		Changes: []types.FileEvent{{
			URI:  mainRegoURI,
			Type: 1, // created
		}},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	waitForViolations(t, "main.rego", []string{"opa-fmt"}, []string{}, timeout, receivedMessages)

	res := must.Return(ls.handleTextDocumentFormatting(ctx, types.DocumentFormattingParams{
		TextDocument: types.TextDocumentIdentifier{URI: mainRegoURI},
	}))(t)

	edits := must.Be[[]types.TextEdit](t, res)
	must.Equal(t, 1, len(edits), "num edits")
	must.Equal(t, types.RangeBetween(1, 0, 2, 0), edits[0].Range, "edit range")
	must.Equal(t, "", edits[0].NewText, "edit new text")
}
