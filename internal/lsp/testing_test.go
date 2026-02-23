package lsp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestLanguageServerTestLocations(t *testing.T) {
	t.Parallel()

	testRegoContents := `package foo_test

test_foo if {
	true
}

test_bar if {
	false
}
`

	files := map[string]string{
		"foo_test.rego": testRegoContents,
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	testRegoURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tempDir, "foo_test.rego"))

	receivedMessages := make(chan map[string]any, defaultBufferedChannelSize)

	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
		switch req.Method {
		case "textDocument/publishDiagnostics":
			// Ignore diagnostics for this test
			return struct{}{}, nil
		case "regal/testLocations":
			return handler.WithParams(req, test.SendsToChannel(receivedMessages))
		default:
			return struct{}{}, nil
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ls, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	go ls.StartTestLocationsWorker(ctx)

	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:  testRegoURI,
			Text: testRegoContents,
		},
	}); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	select {
	case requestData := <-receivedMessages:
		if gotURI, ok := requestData["uri"].(string); !ok || gotURI != testRegoURI {
			t.Fatalf("expected URI %s, got %v", testRegoURI, requestData["uri"])
		}

		locations, ok := requestData["locations"].([]any)
		if !ok {
			t.Fatalf("expected locations to be array, got %T", requestData["locations"])
		}

		if exp, got := 2, len(locations); exp != got {
			t.Fatalf("expected %d test locations, got %d", exp, len(locations))
		}

	case <-timeout.C:
		t.Fatalf("timed out waiting for test locations notification")
	}
}
