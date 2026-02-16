package lsp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

func TestExecuteCommandOpaFmt(t *testing.T) {
	t.Parallel()

	content := `package files

import data.bar
allow if {
1 == 1
    2 == 2
    3 == 4
}



`

	expectedFormattedContent := `package files

import data.bar

allow if {
	1 == 1
	2 == 2
	3 == 4
}
`

	testCases := map[string]struct {
		clientName    string
		expectedEdits []types.TextEdit
	}{
		"generic client": {
			clientName:    "go test",
			expectedEdits: ComputeEdits(content, expectedFormattedContent),
		},
		"IntelliJ client": {
			clientName: "IntelliJ IDEA 2024.2.5",
			expectedEdits: []types.TextEdit{{
				Range:   types.RangeBetween(0, 0, 11, 0),
				NewText: expectedFormattedContent,
			}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			receivedMessages := make(chan types.ApplyWorkspaceEditParams, defaultBufferedChannelSize)

			createWorkspaceApplyEditTestHandler := func(
				t *testing.T,
				receivedMessages chan types.ApplyWorkspaceEditParams,
			) func(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (result any, err error) {
				t.Helper()

				return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
					if req.Method == "workspace/applyEdit" {
						receivedMessages <- must.Return(encoding.JSONUnmarshalTo[types.ApplyWorkspaceEditParams](*req.Params))(t)

						return map[string]any{"applied": true}, nil
					}

					t.Fatalf("unexpected request: %v", req)

					return struct{}{}, nil
				}
			}

			tempDir := t.TempDir()
			clientHandler := createWorkspaceApplyEditTestHandler(t, receivedMessages)
			ls, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

			// set client identifier for this test since we are testing that behavior
			ls.client.Identifier = clients.DetermineIdentifier(tc.clientName)

			// edits are sent to the clinet by the command worker
			go ls.StartCommandWorker(ctx)

			mainRegoURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "main.rego"))
			ls.cache.SetFileContents(mainRegoURI, content)

			// Create command arguments with proper JSON marshaling for Windows backslash escapes
			commandArgs := types.CommandArgs{Target: mainRegoURI}
			argsJSON := must.Return(encoding.JSON().Marshal(commandArgs))(t)

			executeParams := types.ExecuteCommandParams{
				Command:   "regal.fix.opa-fmt",
				Arguments: []any{string(argsJSON)},
			}

			var executeResponse any

			// simulates a manual fmt request from the client
			must.Equal(t, nil, connClient.Call(ctx, "workspace/executeCommand", executeParams, &executeResponse))

			timeout := time.NewTimer(determineTimeout())
			defer timeout.Stop()

			select {
			case applyEditParams := <-receivedMessages:
				must.Equal(t, "Format using opa fmt", applyEditParams.Label, "edit label")
				must.Equal(t, 1, len(applyEditParams.Edit.DocumentChanges), "number of document changes")

				docChange := applyEditParams.Edit.DocumentChanges[0]
				must.Equal(t, mainRegoURI, docChange.TextDocument.URI, "document URI")
				must.Equal(t, len(tc.expectedEdits), len(docChange.Edits), "number of edits")

				for i, expected := range tc.expectedEdits {
					must.Equal(t, expected.Range, docChange.Edits[i].Range, "edit range")
					must.Equal(t, expected.NewText, docChange.Edits[i].NewText, "edit new text")
				}

			case <-timeout.C:
				t.Fatal("timeout waiting for workspace/applyEdit request")
			}
		})
	}
}

func TestExecuteCommandExplorer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	defer func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	receivedNotifications := make(chan map[string]any, defaultBufferedChannelSize)

	createExplorerNotificationTestHandler := func(
		t *testing.T,
		receivedNotifications chan map[string]any,
	) func(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (result any, err error) {
		t.Helper()

		return func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
			if req.Method == "regal/showExplorerResult" {
				if req.Params == nil {
					t.Fatal("expected notification params to be non-nil")
				}

				var notificationData map[string]any
				if err := encoding.JSON().Unmarshal(*req.Params, &notificationData); err != nil {
					t.Fatalf("failed to unmarshal notification params: %s", err)
				}

				receivedNotifications <- notificationData

				return nil, nil
			}

			if req.Method == "workspace/applyEdit" {
				return map[string]any{"applied": true}, nil
			}

			return struct{}{}, nil
		}
	}

	tempDir := t.TempDir()
	clientHandler := createExplorerNotificationTestHandler(t, receivedNotifications)
	ls, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	// Set client identifier to VSCode so it uses the notification approach
	ls.client.Identifier = clients.IdentifierVSCode

	go ls.StartCommandWorker(ctx)

	content := `package test

allow if {
	1 == 1
}
`

	mainRegoURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "test.rego"))
	ls.cache.SetFileContents(mainRegoURI, content)

	executeParams := types.ExecuteCommandParams{
		Command: "regal.explorer",
		Arguments: []any{
			map[string]any{
				"target":      mainRegoURI,
				"strict":      false,
				"annotations": false,
				"print":       false,
				"format":      true,
			},
		},
	}

	var executeResponse any
	must.Equal(t, nil, connClient.Call(ctx, "workspace/executeCommand", executeParams, &executeResponse))

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	select {
	case notification := <-receivedNotifications:
		if _, ok := notification["stages"]; !ok {
			t.Fatal("expected notification to contain 'stages' field")
		}

		stages := must.Be[[]any](t, notification["stages"])
		must.NotEqual(t, 0, len(stages), "expected at least one compilation stage")

		firstStage := must.Be[map[string]any](t, stages[0])
		for _, field := range []string{"name", "output", "error"} {
			if _, ok := firstStage[field]; !ok {
				t.Fatalf("expected stage to have '%s' field", field)
			}
		}
	case <-timeout.C:
		t.Fatal("timeout waiting for regal/showExplorerResult notification")
	}
}
