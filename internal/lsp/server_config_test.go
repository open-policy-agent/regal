package lsp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// TestLanguageServerParentDirConfig tests that regal config is loaded as it is for the
// Regal CLI, and that config files in a parent directory are loaded correctly
// even when the workspace is a child directory.
func TestLanguageServerParentDirConfig(t *testing.T) {
	t.Parallel()

	mainRegoContents := `package main

import data.test
allow := true
`

	childDirName := "child"

	files := map[string]string{
		childDirName + mainRegoFileName: mainRegoContents,
		".regal/config.yaml": `rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  style:
    opa-fmt:
      level: error
`,
	}

	// childDir will be the directory that the client is using as its workspace
	tempDir := testutil.TempDirectoryOf(t, files)

	receivedMessages := createMessageChannels(files)
	clientHandler := createPublishDiagnosticsHandler(t, log.NewLogger(log.LevelDebug, t.Output()), receivedMessages)

	ls, _, ctx := createAndInitServer(t, tempDir, clientHandler)

	ls.StartConfigWorker(ctx)

	if got, exp := ls.getWorkspaceRootURI(), uri.FromPath(ls.getClient().Identifier, tempDir); exp != got {
		t.Fatalf("expected client root URI to be %s, got %s", exp, got)
	}

	timeout := time.NewTimer(determineTimeout())
	defer timeout.Stop()

	waitForViolations(t, "main.rego", []string{"opa-fmt"}, []string{}, timeout, receivedMessages)

	// User updates config file contents in parent directory that is not
	// part of the workspace
	newConfigContents := `rules:
  idiomatic:
    directory-package-mismatch:
      level: ignore
  style:
    opa-fmt:
      level: ignore
`

	must.WriteFile(t, filepath.Join(tempDir, ".regal", "config.yaml"), []byte(newConfigContents))

	// validate that the client received a new, empty diagnostics notification for the file
	timeout.Reset(determineTimeout())

	waitForViolations(t, "main.rego", []string{}, []string{}, timeout, receivedMessages)
}
