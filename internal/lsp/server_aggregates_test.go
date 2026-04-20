package lsp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// Test to ensure that annotations are parsed correctly.
func TestRulesWithMetadataNotReportedForMissingMeta(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego": `# METADATA
# title: foo
package foo
`,
		"bar.rego": `# METADATA
# title: foo
package bar
`,
		".regal/config.yaml": `rules:
  custom:
    missing-metadata:
      level: error
  idiomatic:
    directory-package-mismatch:
      level: ignore
`,
	}

	receivedMessages := createMessageChannels(files)
	logger := log.NewLogger(log.LevelDebug, t.Output())
	clientHandler := createPublishDiagnosticsHandler(t, logger, receivedMessages)

	tempDir := testutil.TempDirectoryOf(t, files)

	ls, _, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	// Wait for custom config to load with directory-package-mismatch set to ignore
	// and missing-metadata set to error
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
					// Also verify missing-metadata is set to error
					if mmRule, ok := cfg.Rules["custom"]["missing-metadata"]; ok && mmRule.Level == "error" {
						break
					}
				}
			}
		}
	}

	timeout.Reset(determineTimeout())

	// no missing-metadata
	waitForViolations(t, "foo.rego", []string{}, []string{}, timeout, receivedMessages)
}

func TestLanguageServerAggregateViolationFixedAndReintroducedInUnviolatingFileChange(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego": `package foo

import rego.v1

import data.bax # initially unresolved-import

variable = "string" # use-assignment-operator
`,
		"bar.rego": `package bar

import rego.v1
`,
		".regal/config.yaml": ``,
	}

	logger := log.NewLogger(log.LevelDebug, t.Output())
	tempDir := testutil.TempDirectoryOf(t, files)
	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, logger, receivedMessages)

	ls, connClient, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	// wait for foo.rego to have the correct violations
	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	waitForViolations(
		t,
		"foo.rego",
		[]string{"unresolved-import", "use-assignment-operator"},
		[]string{},
		timeout,
		receivedMessages,
	)

	// update the contents of the bar.rego file to address the unresolved-import
	notifyDocumentChange(
		t,
		connClient,
		uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego")),
		`package bax # package imported in foo.rego

import rego.v1
`)

	// wait for foo.rego to have the correct violations
	timeout.Reset(determineTimeout())

	waitForViolations(
		t,
		"foo.rego",
		[]string{"use-assignment-operator"},
		[]string{"unresolved-import"},
		timeout,
		receivedMessages,
	)

	// update the contents of the bar.rego to bring back the violation
	barURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(tempDir, "bar.rego"))
	notifyDocumentChange(t, connClient, barURI, `package bar # original package to bring back the violation

import rego.v1
`)

	// check the violation is back
	timeout.Reset(determineTimeout())

	waitForViolations(
		t,
		"foo.rego",
		[]string{"unresolved-import", "use-assignment-operator"},
		[]string{},
		timeout,
		receivedMessages,
	)
}
