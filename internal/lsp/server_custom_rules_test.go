package lsp

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestLanguageServerCustomRule(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		".regal/config.yaml": "",
		".regal/rules/custom.rego": `# METADATA
# description: No var named "custom"
# schemas:
# - input: schema.regal.ast
package custom.regal.rules.naming.custom

import data.regal.ast
import data.regal.result

report contains violation if {
	some i
	var := ast.found.vars[i][_][_]

	lower(var.value) == "custom"

	ast.is_output_var(input.rules[to_number(i)], var)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}

`,
		"example/foo.rego": `package example

allow if {
	custom := 1
	1 == 2
}
`,
	}

	tempDir := testutil.TempDirectoryOf(t, files)

	logger := log.NewLogger(log.LevelDebug, t.Output())
	messages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, logger, messages)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, connClient := createAndInitServer(t, ctx, tempDir, clientHandler)

	// send textDocument/didOpen notification to trigger diagnostics
	if err := connClient.Notify(ctx, "textDocument/didOpen", types.DidOpenTextDocumentParams{
		TextDocument: types.TextDocumentItem{
			URI:  uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "example", "foo.rego")),
			Text: files["example/foo.rego"],
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didOpen notification: %s", err)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// wait for diagnostics to be published file with the custom violation set
	for success := false; !success; {
		select {
		case violations := <-messages["foo.rego"]:
			if !slices.Contains(violations, "custom") {
				t.Logf("waiting for violations to contain \"custom\", have: %v", violations)

				continue
			}

			success = true

		case <-timeout.C:
			t.Fatalf("timed out waiting for foo.rego diagnostics to be sent")
		}
	}
}
