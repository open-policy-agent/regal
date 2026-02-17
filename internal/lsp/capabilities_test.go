package lsp

import (
	"context"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestInitializeExperimentalCapabilities(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tempDir := t.TempDir()

	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (any, error) {
		return struct{}{}, nil
	}

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	request := types.InitializeParams{
		RootURI:    uri.FromPath(clients.IdentifierGeneric, tempDir),
		ClientInfo: types.ClientInfo{Name: "go test"},
	}

	var response types.InitializeResult
	testutil.NoErr(connClient.Call(ctx, "initialize", request, &response))(t)

	if response.Capabilities.Experimental == nil {
		t.Fatal("expected experimental capabilities to be non-nil")
	}

	if !response.Capabilities.Experimental.ExplorerProvider {
		t.Error("expected explorerProvider to be true")
	}

	if !response.Capabilities.Experimental.InlineEvalProvider {
		t.Error("expected inlineEvalProvider to be true")
	}

	if !response.Capabilities.Experimental.DebugProvider {
		t.Error("expected debugProvider to be true")
	}
}
