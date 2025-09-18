package lsp

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
)

const (
	mainRegoFileName = "/main.rego"
	fileURIScheme    = "file://"
	// defaultTimeout is set based on the investigation done as part of
	// https://github.com/open-policy-agent/regal/issues/931. 20 seconds is 10x the
	// maximum time observed for an operation to complete.
	defaultTimeout             = 20 * time.Second
	defaultBufferedChannelSize = 5
)

// determineTimeout returns a timeout duration based on whether
// the test suite is running with race detection, if so, a more permissive
// timeout is used.
func determineTimeout() time.Duration {
	if isRaceEnabled() {
		// based on the upper bound here, 20x slower
		// https://go.dev/doc/articles/race_detector#Runtime_Overheads
		return defaultTimeout * 20
	}

	return defaultTimeout
}

func createAndInitServer(
	t *testing.T,
	ctx context.Context,
	tempDir string,
	clientHandler func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error),
) (
	*LanguageServer,
	*jsonrpc2.Conn,
) {
	t.Helper()

	// This is set due to eventing being so slow in go test -race that we
	// get flakes. TODO, work out how to avoid needing this in lsp tests.
	pollingInterval := time.Duration(0)
	if isRaceEnabled() {
		pollingInterval = 10 * time.Second
	}

	logger := log.NewLogger(log.LevelDebug, t.Output())

	// set up the server and client connections
	ls := NewLanguageServer(ctx, &LanguageServerOptions{Logger: logger, WorkspaceDiagnosticsPoll: pollingInterval})

	go ls.StartDiagnosticsWorker(ctx)
	go ls.StartConfigWorker(ctx)

	netConnServer, netConnClient := net.Pipe()

	connServer := jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(netConnServer, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(ls.Handle),
	)

	connClient := jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(netConnClient, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(clientHandler),
	)

	go func() {
		<-ctx.Done()
		// we need only close the pipe connections as the jsonrpc2.Conn accept
		// the ctx
		_ = netConnClient.Close()
		_ = netConnServer.Close()
	}()

	ls.SetConn(connServer)

	// a blank tempDir means no workspace root was required.
	rootURI := ""
	if tempDir != "" {
		rootURI = uri.FromPath(clients.IdentifierGeneric, tempDir)
	}

	request := types.InitializeParams{
		RootURI:    rootURI,
		ClientInfo: types.ClientInfo{Name: "go test"},
	}

	var response types.InitializeResult
	if err := connClient.Call(ctx, "initialize", request, &response); err != nil {
		t.Fatalf("failed to initialize: %s", err)
	}

	// 2. Client sends initialized notification
	// no response to the call is expected
	if err := connClient.Call(ctx, "initialized", struct{}{}, nil); err != nil {
		t.Fatalf("failed to complete initialized: %v", err)
	}

	return ls, connClient
}

func createPublishDiagnosticsHandler(
	t *testing.T,
	logger io.Writer,
	messages map[string]chan []string,
) func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	t.Helper()

	return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		if req.Method != methodTdPublishDiagnostics {
			fmt.Fprintln(logger, "createClientHandler: unexpected request method:", req.Method)

			return struct{}{}, nil
		}

		return handler.WithParams(req, func(params types.FileDiagnostics) (any, error) {
			violations := make([]string, len(params.Items))
			for i, item := range params.Items {
				violations[i] = item.Code
			}

			slices.Sort(violations)

			fileBase := filepath.Base(params.URI)
			fmt.Fprintln(logger, "createPublishDiagnosticsHandler: queue", fileBase, len(messages[fileBase]))

			select {
			case messages[fileBase] <- violations:
			case <-time.After(1 * time.Second):
				t.Fatalf("timeout writing to messages channel for %s", fileBase)
			}

			return struct{}{}, nil
		})
	}
}

func createMessageChannels(files map[string]string) map[string]chan []string {
	messages := make(map[string]chan []string)
	for _, file := range util.Keys(files) {
		messages[filepath.Base(file)] = make(chan []string, 10)
	}

	return messages
}

func testRequestDataCodes(t *testing.T, requestData types.FileDiagnostics, fileURI string, codes []string) bool {
	t.Helper()

	if requestData.URI != fileURI {
		t.Log("expected diagnostics to be sent for", fileURI, "got", requestData.URI)

		return false
	}

	// Extract the codes from requestData.Items
	requestCodes := make([]string, len(requestData.Items))
	for i, item := range requestData.Items {
		requestCodes[i] = item.Code
	}

	// Sort both slices
	slices.Sort(requestCodes)
	slices.Sort(codes)

	if !slices.Equal(requestCodes, codes) {
		t.Logf("waiting for items: %v, got: %v", codes, requestCodes)

		return false
	}

	t.Logf("got expected items")

	return true
}
