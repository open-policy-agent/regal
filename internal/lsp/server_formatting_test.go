package lsp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	// set up the server and client connections
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		t.Fatalf("unexpected request: %v", req)

		return struct{}{}, nil
	}

	tempDir := t.TempDir()
	ls, _ := createAndInitServer(t, ctx, tempDir, clientHandler)
	mainRegoURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "main", "main.rego"))

	// Simple as possible — opa fmt should just remove a newline
	ls.cache.SetFileContents(mainRegoURI, "package main\n\n")

	res := must.Return(ls.handleTextDocumentFormatting(ctx, types.DocumentFormattingParams{
		TextDocument: types.TextDocumentIdentifier{URI: mainRegoURI},
		Options:      types.FormattingOptions{},
	}))(t)

	edits := must.Be[[]types.TextEdit](t, res)
	must.Equal(t, 1, len(edits), "num edits")
	must.Equal(t, types.RangeBetween(1, 0, 2, 0), edits[0].Range, "edit range")
	must.Equal(t, "", edits[0].NewText, "edit new text")
}
