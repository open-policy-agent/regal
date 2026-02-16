// Package lsp exposes certain internals of the lsp server for use with web-based clients to the Regal LSP.
//
// Use with care, interfaces may change. This package is considered experimental!
package lsp

import (
	"context"
	"os"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	jsonrpc2_ws "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/open-policy-agent/regal/internal/lsp"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/pkg/config"
)

// Handle represents a single active LSP session.
type Handle struct {
	conn *jsonrpc2.Conn
	ls   *lsp.LanguageServer
}

// New opens a new language server session on the provided websocket connection `ws`.
// Use the `*config.Config` argument to control the LSP session's config.
func New(ctx context.Context, ws *websocket.Conn, c *config.Config) (*Handle, error) {
	opts := lsp.LanguageServerOptions{
		Logger:       log.NewLogger(log.LevelOff, os.Stderr),
		FeatureFlags: lsp.DefaultServerFeatureFlags(),
	}
	if os.Getenv("REGAL_DEBUG") != "" {
		opts.Logger.SetLevel(log.LevelDebug)
	}

	ls := lsp.NewLanguageServerMinimal(ctx, &opts, c)
	jconn := jsonrpc2.NewConn(
		ctx,
		jsonrpc2_ws.NewObjectStream(ws),
		jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(ls.Handle)),
	)
	ls.SetConn(jconn)

	go ls.StartDiagnosticsWorker(ctx)
	go ls.StartHoverWorker(ctx)
	go ls.StartCommandWorker(ctx)

	return &Handle{conn: jconn, ls: ls}, nil
}

// Wait waits for the client to finish its exchange. It aborts if ctx is done.
func (h *Handle) Wait(ctx context.Context) error {
	select {
	case <-h.conn.DisconnectNotify():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the underlying connection.
func (h *Handle) Close() error {
	return h.conn.Close()
}
