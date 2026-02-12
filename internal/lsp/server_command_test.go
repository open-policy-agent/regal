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
	"github.com/open-policy-agent/regal/internal/testutil"
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
						requestData, err := encoding.JSONUnmarshalTo[types.ApplyWorkspaceEditParams](*req.Params)
						if err != nil {
							t.Fatalf("failed to unmarshal applyEdit params: %s", err)
						}

						receivedMessages <- requestData

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
			argsJSON := testutil.Must(encoding.JSON().Marshal(commandArgs))(t)

			executeParams := types.ExecuteCommandParams{
				Command:   "regal.fix.opa-fmt",
				Arguments: []any{string(argsJSON)},
			}

			var executeResponse any

			// simulates a manual fmt request from the client
			testutil.NoErr(connClient.Call(ctx, "workspace/executeCommand", executeParams, &executeResponse))(t)

			timeout := time.NewTimer(determineTimeout())
			defer timeout.Stop()

			select {
			case applyEditParams := <-receivedMessages:
				if applyEditParams.Label != "Format using opa fmt" {
					t.Fatalf("expected label 'Format using opa fmt', got %s", applyEditParams.Label)
				}

				if len(applyEditParams.Edit.DocumentChanges) != 1 {
					t.Fatalf("expected 1 document change, got %d", len(applyEditParams.Edit.DocumentChanges))
				}

				docChange := applyEditParams.Edit.DocumentChanges[0]
				if docChange.TextDocument.URI != mainRegoURI {
					t.Fatalf("expected URI %s, got %s", mainRegoURI, docChange.TextDocument.URI)
				}

				if len(docChange.Edits) != len(tc.expectedEdits) {
					t.Fatalf("expected %d edits, got %d", len(tc.expectedEdits), len(docChange.Edits))
				}

				for i, expected := range tc.expectedEdits {
					actual := docChange.Edits[i]
					if actual.Range != expected.Range || actual.NewText != expected.NewText {
						t.Fatalf("edit %d mismatch:\nexpected: %v\nactual:   %v", i, expected, actual)
					}
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

	testutil.NoErr(connClient.Call(ctx, "workspace/executeCommand", executeParams, &executeResponse))(t)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	select {
	case notification := <-receivedNotifications:
		if _, ok := notification["stages"]; !ok {
			t.Fatal("expected notification to contain 'stages' field")
		}

		stages, ok := notification["stages"].([]any)
		if !ok {
			t.Fatalf("expected 'stages' to be a slice, got %T", notification["stages"])
		}

		if len(stages) == 0 {
			t.Fatal("expected at least one compilation stage")
		}

		firstStage, ok := stages[0].(map[string]any)
		if !ok {
			t.Fatalf("expected first stage to be a map, got %T", stages[0])
		}

		if _, ok := firstStage["name"]; !ok {
			t.Fatal("expected stage to have 'name' field")
		}

		if _, ok := firstStage["output"]; !ok {
			t.Fatal("expected stage to have 'output' field")
		}

		if _, ok := firstStage["error"]; !ok {
			t.Fatal("expected stage to have 'error' field")
		}

	case <-timeout.C:
		t.Fatal("timeout waiting for regal/showExplorerResult notification")
	}
}
