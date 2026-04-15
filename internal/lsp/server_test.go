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

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/connection"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
)

const (
	mainRegoFileName = "/main.rego"
	// defaultTimeout is set based on the investigation done as part of
	// https://github.com/open-policy-agent/regal/issues/931. 20 seconds is 10x the
	// maximum time observed for an operation to complete.
	defaultTimeout             = 20 * time.Second
	defaultBufferedChannelSize = 5
)

type receivedMessagesMap map[string]chan []string

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

// drainMessages drains all pending messages from a channel.
// This is useful in tests to clear buffered messages and avoid race conditions
// where old messages are consumed instead of waiting for new ones.
// It's common for messages to build up in the race detector when things are
// running very slowly.
func drainMessages[T any](ch chan T) {
	for {
		select {
		case <-ch:
			// Keep draining
		default:
			return
		}
	}
}

func createAndInitServer(t *testing.T, tempDir string, clientHandler connection.HandlerFunc) (
	*LanguageServer,
	*jsonrpc2.Conn,
	context.Context,
) {
	t.Helper()

	return createAndInitServerWithClientName(t, tempDir, clientHandler, "go test")
}

func createAndInitServerWithClientName(
	t *testing.T,
	tempDir string,
	clientHandler connection.HandlerFunc,
	clientName string,
) (
	*LanguageServer,
	*jsonrpc2.Conn,
	context.Context,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	// This is set due to eventing being so slow in go test -race that we
	// get flakes. TODO, work out how to avoid needing this in lsp tests.
	pollingInterval := time.Duration(0)
	if isRaceEnabled() {
		pollingInterval = 10 * time.Second
	}

	logger := log.NewLogger(log.LevelDebug, t.Output())

	// set up the server and client connections
	ls := NewLanguageServer(ctx, &LanguageServerOptions{
		Logger:                   logger,
		WorkspaceDiagnosticsPoll: pollingInterval,
		FeatureFlags:             DefaultServerFeatureFlags(),
	})

	ls.StartDiagnosticsWorker(ctx)
	ls.StartConfigWorker(ctx)
	ls.StartTestLocationsWorker(ctx)
	ls.StartCommandWorker(ctx)

	// Not started automatically:
	// - StartTemplateWorker: Manually started where needed to test for ordering bugs
	// - StartWorkspaceStateWorker: Only needed for long-running tests monitoring workspace changes
	// - StartQueryCacheWorker: Only needed in dev mode (REGAL_BUNDLE_PATH set)
	// - StartWebServer: Not used in tests

	netConnServer, netConnClient := net.Pipe()

	connServer := connection.New(ctx, netConnServer, ls.Handle)
	connClient := connection.New(ctx, netConnClient, clientHandler)

	// Register cleanup to cancel context, wait for workers, and close connections.
	// t.Cleanup runs after test completes (including any defers in the test).
	t.Cleanup(func() {
		cancel()

		// Use context.Background() to ensure a valid context for shutdown,
		// independent of the test's lifecycle.
		//nolint:usetesting
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		_ = ls.Shutdown(shutdownCtx)

		// Close jsonrpc2 connections before closing the underlying pipes
		// to prevent "read/write on closed pipe" errors from readMessages()
		_ = connServer.Close()
		_ = connClient.Close()

		_ = netConnClient.Close()
		_ = netConnServer.Close()
	})

	ls.SetConn(connServer)

	// Determine client identifier from name for URI construction
	clientIdentifier := clients.DetermineIdentifier(clientName)

	// a blank tempDir means no workspace root was required.
	rootURI := ""
	if tempDir != "" {
		rootURI = uri.FromPath(clientIdentifier, tempDir)
	}

	request := types.InitializeParams{
		RootURI:    rootURI,
		ClientInfo: types.ClientInfo{Name: clientName},
		InitializationOptions: &types.InitializationOptions{
			EnableDebugCodelens:       new(true),
			EnableExplorer:            new(true),
			EvalCodelensDisplayInline: new(true),
			EnableServerTesting:       new(true),
		},
	}

	var response types.InitializeResult
	testutil.NoErr(connClient.Call(ctx, "initialize", request, &response))(t)

	// 2. Client sends initialized notification
	// no response to the call is expected
	testutil.NoErr(connClient.Call(ctx, "initialized", struct{}{}, nil))(t)

	// wait for the server to complete the start up process to avoid races
	// where the initial workspace loading and linting races with requests
	// sent in tests. This is not an issue for diagnostics, since the file jobs
	// are only run after the initializationGate is closed, but the server can
	// still process other requests like apply edits and commands which
	// can cause inconsistent filecontents in the cache.
	<-ls.initializationGate

	return ls, connClient, ctx
}

func createPublishDiagnosticsHandler(
	t *testing.T,
	out io.Writer,
	receivedMessages receivedMessagesMap,
) connection.HandlerFunc {
	t.Helper()

	return func(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
		if req.Method != methodTdPublishDiagnostics {
			// Check context before writing to test output to avoid panic if test has completed
			if ctx.Err() == nil {
				fmt.Fprintln(out, "createClientHandler: unexpected request method:", req.Method)
			}

			return struct{}{}, nil
		}

		return handler.WithParams(req, func(params types.FileDiagnostics) (any, error) {
			violations := make([]string, len(params.Items))
			for i, item := range params.Items {
				violations[i] = item.Code
			}

			fileBase := filepath.Base(params.URI)
			// Check context before writing to test output to avoid panic if test has completed
			if ctx.Err() == nil {
				fmt.Fprintln(out, "createPublishDiagnosticsHandler: queue", fileBase, len(receivedMessages[fileBase]))
			}

			select {
			case receivedMessages[fileBase] <- util.Sorted(violations):
			case <-time.After(1 * time.Second):
				t.Fatalf("timeout writing to receivedMessages channel for %s", fileBase)
			}

			return struct{}{}, nil
		})
	}
}

func createMessageChannels(files map[string]string) receivedMessagesMap {
	receivedMessages := make(receivedMessagesMap, len(files))
	for _, file := range util.MapKeys(files, filepath.Base) {
		receivedMessages[file] = make(chan []string, 10)
	}

	return receivedMessages
}

func testRequestDataCodes(t *testing.T, requestData types.FileDiagnostics, fileURI string, codes []string) bool {
	t.Helper()

	if requestData.URI != fileURI {
		t.Log("expected diagnostics to be sent for", fileURI, "got", requestData.URI)

		return false
	}

	// If codes is nil, we just want any diagnostics for this file
	if codes == nil {
		t.Logf("got diagnostics for %s (not checking specific codes)", fileURI)

		return true
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

func TestPositionToOffset(t *testing.T) {
	t.Parallel()

	text := "line1\nline2\nline3"

	for line := range uint(2) {
		for char := range uint(5) {
			pos := types.Position{Line: line, Character: char}
			exp := line*6 + char
			got := util.SafeIntToUint(positionToOffset(text, pos))

			if exp != got {
				t.Fatalf("expected offset for line %d char %d to be %d, got %d", line, char, exp, got)
			}
		}
	}
}

func waitForDiagnostics(
	t *testing.T,
	receivedMessages <-chan types.FileDiagnostics,
	fileURI string,
	expectedCodes []string,
	timeout *time.Timer,
) {
	t.Helper()

	for {
		select {
		case requestData := <-receivedMessages:
			if testRequestDataCodes(t, requestData, fileURI, expectedCodes) {
				return
			}
		case <-timeout.C:
			t.Fatalf(
				"timed out waiting for diagnostics for %s with codes %v",
				fileURI,
				expectedCodes,
			)
		}
	}
}

func notifyDocumentChange(t *testing.T, connClient *jsonrpc2.Conn, fileURI, newContents string) {
	t.Helper()

	err := connClient.Notify(t.Context(), "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument:   types.VersionedTextDocumentIdentifier{URI: fileURI},
		ContentChanges: []types.TextDocumentContentChangeEvent{{Text: newContents}},
	}, nil)
	if err != nil {
		t.Fatalf("failed to send didChange notification: %v", err)
	}
}

// waitForViolations waits for violations to match the expected state.
// wantPresent: violation codes that must be present in diagnostics
// wantAbsent: violation codes that must be absent from diagnostics
// Pass empty slices to check for no violations at all.
func waitForViolations(
	t *testing.T,
	key string,
	wantPresent,
	wantAbsent []string,
	timeout *time.Timer,
	receivedMessages receivedMessagesMap,
) {
	t.Helper()

	for success := false; !success; {
		select {
		case violations := <-receivedMessages[key]:
			allMatch := true

			// Check all rules that should be present
			for _, rule := range wantPresent {
				if !slices.Contains(violations, rule) {
					t.Logf("waiting for violations to contain %s", rule)

					allMatch = false

					break
				}
			}

			if !allMatch {
				continue
			}

			// Check all rules that should be absent
			for _, rule := range wantAbsent {
				if slices.Contains(violations, rule) {
					t.Logf("waiting for violations to not contain %s", rule)

					allMatch = false

					break
				}
			}

			if !allMatch {
				continue
			}

			// If both slices are empty, check that violations is empty
			if len(wantPresent) == 0 && len(wantAbsent) == 0 {
				if len(violations) > 0 {
					t.Logf("waiting for violations to be empty for %s, have: %v", key, violations)

					continue
				}
			}

			success = true
		case <-timeout.C:
			t.Fatalf("timed out waiting for violations - want present: %v, want absent: %v", wantPresent, wantAbsent)
		}
	}
}
