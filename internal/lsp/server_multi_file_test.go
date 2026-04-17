package lsp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// TestLanguageServerMultipleFiles tests that changes to multiple files are handled correctly. When there are multiple
// files in the workspace, the diagnostics worker also processes aggregate violations, there are also changes to when
// workspace diagnostics are run, this test validates that the correct diagnostics are sent to the client in this
// scenario.
func TestLanguageServerMultipleFiles(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"authz.rego": `package authz

import rego.v1

import data.admins.users

default allow := false

allow if input.user in users
`,
		"admins.rego": `package admins

import rego.v1

users = {"alice", "bob"}
`,
		"ignored/foo.rego": `package ignored

foo = 1
`,
		".regal/config.yaml": `
rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
ignore:
  files:
    - ignored/*.rego
`,
	}

	// set up the workspace content with some example rego and regal config
	tempDir := testutil.TempDirectoryOf(t, files)
	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), receivedMessages)

	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// Wait for custom config to load with directory-package-mismatch set to ignore
	select {
	case <-timeout.C:
		t.Fatalf("timed out waiting for server to load config")
	default:
		for {
			time.Sleep(testPollInterval)

			cfg := ls.getLoadedConfig()
			if cfg != nil {
				// Verify directory-package-mismatch is ignored
				if rule, ok := cfg.Rules["idiomatic"]["directory-package-mismatch"]; ok && rule.Level == "ignore" {
					break
				}
			}
		}
	}

	timeout.Reset(determineTimeout())

	// validate that the client received a diagnostics notification for authz.rego
	waitForViolations(t, "authz.rego", []string{"prefer-package-imports"}, []string{}, timeout, receivedMessages)

	// validate that the client received a diagnostics notification for admins.rego
	timeout.Reset(determineTimeout())

	waitForViolations(t, "admins.rego", []string{"use-assignment-operator"}, []string{}, timeout, receivedMessages)

	// 3. Client sends textDocument/didChange notification with new contents
	// for authz.rego no response to the call is expected
	if err := connClient.Notify(ctx, "textDocument/didChange", types.DidChangeTextDocumentParams{
		TextDocument: types.VersionedTextDocumentIdentifier{
			URI: uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "authz.rego")),
		},
		ContentChanges: []types.TextDocumentContentChangeEvent{
			{
				Text: `package authz

import rego.v1

import data.admins # fixes prefer-package-imports

default allow := false

# METADATA
# description: Allow only admins
# entrypoint: true # fixes no-defined-entrypoint
allow if input.user in admins.users
`,
			},
		},
	}, nil); err != nil {
		t.Fatalf("failed to send didChange notification: %s", err)
	}

	// authz.rego should now have no violations
	timeout.Reset(determineTimeout())

	waitForViolations(t, "authz.rego", []string{}, []string{}, timeout, receivedMessages)
}
