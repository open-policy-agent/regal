package lsp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
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

			mainRegoURI := fileURIScheme + filepath.Join(tempDir, "main.rego")
			ls.cache.SetFileContents(mainRegoURI, content)

			executeParams := types.ExecuteCommandParams{
				Command:   "regal.fix.opa-fmt",
				Arguments: []any{`{"target":"` + mainRegoURI + `"}`},
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
