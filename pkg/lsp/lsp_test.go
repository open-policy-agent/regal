package lsp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	jsonrpc2_ws "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/open-policy-agent/regal/pkg/lsp"
)

func TestConcurrentLSPInitializationAndDiagnostics(t *testing.T) {
	t.Parallel()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handle, err := lsp.New(r.Context(), ws, nil)
		if err != nil {
			_ = ws.Close()

			return
		}

		_ = handle.Wait(r.Context())
		_ = handle.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	baseTempDir := t.TempDir()

	const concurrency = 6
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := range concurrency {
		go func(sessionID int) {
			defer wg.Done()

			sessionDir := filepath.Join(baseTempDir, fmt.Sprintf("session-%d", sessionID))
			if err := os.MkdirAll(sessionDir, 0o755); err != nil {
				t.Errorf("session %d mkdir error: %v", sessionID, err)

				return
			}
			rootURI := "file://" + filepath.ToSlash(sessionDir)
			policyURI := rootURI + "/policy.rego"

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
			if err != nil {
				t.Errorf("session %d dial error: %v", sessionID, err)

				return
			}
			defer ws.Close()

			receivedDiagnostics := make(chan struct{}, 10)
			clientHandler := jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
				if req.Method == "textDocument/publishDiagnostics" {
					select {
					case receivedDiagnostics <- struct{}{}:
					default:
					}
				}

				return struct{}{}, nil
			})

			clientConn := jsonrpc2.NewConn(
				ctx,
				jsonrpc2_ws.NewObjectStream(ws),
				clientHandler,
			)
			defer clientConn.Close()

			var initRes any
			if err := clientConn.Call(ctx, "initialize", map[string]any{
				"rootUri":      rootURI,
				"capabilities": map[string]any{},
			}, &initRes); err != nil {
				t.Errorf("session %d initialize failed: %v", sessionID, err)

				return
			}

			if err := clientConn.Notify(ctx, "initialized", map[string]any{}); err != nil {
				t.Errorf("session %d initialized notification failed: %v", sessionID, err)

				return
			}

			if err := clientConn.Notify(ctx, "textDocument/didOpen", map[string]any{
				"textDocument": map[string]any{
					"uri":        policyURI,
					"languageId": "rego",
					"version":    1,
					"text":       "package policy\n\ndefault allow := false\n\nallow if {\n\t1 == 1\n}\n",
				},
			}); err != nil {
				t.Errorf("session %d didOpen failed: %v", sessionID, err)

				return
			}

			// Wait briefly for diagnostics worker to process
			select {
			case <-receivedDiagnostics:
			case <-time.After(2 * time.Second):
			}

			_ = clientConn.Notify(ctx, "textDocument/didChange", map[string]any{
				"textDocument": map[string]any{
					"uri":     policyURI,
					"version": 2,
				},
				"contentChanges": []map[string]any{
					{"text": "package policy\n\ndefault allow := true\n"},
				},
			})

			time.Sleep(100 * time.Millisecond)

			var shutdownRes any
			_ = clientConn.Call(ctx, "shutdown", nil, &shutdownRes)
			_ = clientConn.Notify(ctx, "exit", nil)
		}(i)
	}

	wg.Wait()
}

func TestImmediateClientMessageOnConnect(t *testing.T) {
	t.Parallel()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handle, err := lsp.New(r.Context(), ws, nil)
		if err != nil {
			_ = ws.Close()

			return
		}

		_ = handle.Wait(r.Context())
		_ = handle.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	const iterations = 10
	for i := range iterations {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		if err != nil {
			cancel()
			t.Fatalf("iteration %d dial error: %v", i, err)
		}

		clientConn := jsonrpc2.NewConn(
			ctx,
			jsonrpc2_ws.NewObjectStream(ws),
			jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (any, error) {
				return struct{}{}, nil
			}),
		)

		sessionDir := filepath.Join(t.TempDir(), fmt.Sprintf("iter-%d", i))
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			cancel()
			t.Fatalf("iteration %d mkdir error: %v", i, err)
		}
		rootURI := "file://" + filepath.ToSlash(sessionDir)

		var initRes any
		// Immediately send initialize to trigger concurrent SetConn vs Handle
		err = clientConn.Call(ctx, "initialize", map[string]any{
			"rootUri":      rootURI,
			"capabilities": map[string]any{},
		}, &initRes)
		if err != nil {
			t.Errorf("iteration %d immediate initialize call failed: %v", i, err)
		}

		var shutdownRes any
		_ = clientConn.Call(ctx, "shutdown", nil, &shutdownRes)
		_ = clientConn.Notify(ctx, "exit", nil)

		_ = clientConn.Close()
		_ = ws.Close()
		cancel()
	}
}
