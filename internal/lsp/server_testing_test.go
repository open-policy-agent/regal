package lsp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestHandleRunTest(t *testing.T) {
	t.Parallel()

	testRegoContents := `package bar_test

test_with_output if {
	print("hello from test")
	true
}
`

	files := map[string]string{
		"bar_test.rego": testRegoContents,
	}

	tempDir := testutil.TempDirectoryOf(t, files)
	testRegoURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tempDir, "bar_test.rego"))

	clientHandler := func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
		switch req.Method {
		case "textDocument/publishDiagnostics":
			return struct{}{}, nil
		case "regal/testLocations":
			return struct{}{}, nil
		default:
			return struct{}{}, nil
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:  testRegoURI,
			Text: testRegoContents,
		},
	}); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	params := types.RunTestsParams{
		URI:     testRegoURI,
		Package: "data.bar_test",
		Name:    "test_with_output",
	}

	var result []tester.Result
	if err := connClient.Call(ctx, "regal/runTests", params, &result); err != nil {
		t.Fatalf("failed to call regal/runTests: %s", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 test result, got %d", len(result))
	}

	if !result[0].Pass() {
		t.Errorf("expected test to pass, but it failed")
	}

	if len(result[0].Output) == 0 {
		t.Errorf("expected output to be captured, but it was empty")
	}
}
