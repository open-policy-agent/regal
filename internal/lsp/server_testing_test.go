package lsp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/test"
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

	receivedMessages := make(chan types.FileDiagnostics, defaultBufferedChannelSize)
	clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedMessages))

	_, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:  testRegoURI,
			Text: testRegoContents,
		},
	}); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	// Wait for diagnostics to be published (ensures file is parsed before running tests)
	// We don't care about the actual diagnostic codes, just that parsing completed
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	select {
	case <-receivedMessages:
		// Diagnostics received, file is parsed
	case <-timeout.C:
		t.Fatal("timeout waiting for diagnostics")
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
