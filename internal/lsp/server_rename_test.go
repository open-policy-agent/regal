package lsp

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestLanguageServerFixRenameParams(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testutil.MustMkdirAll(t, tmpDir, "workspace", "foo", "bar")

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierVSCode
	ls.workspaceRootURI = fmt.Sprintf("file://%s/workspace", tmpDir)
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{
			"idiomatic": {
				"directory-package-mismatch": config.Rule{
					Level: "ignore",
					Extra: map[string]any{
						"exclude-test-suffix": true,
					},
				},
			},
		},
	}

	fileURI := ls.workspaceRootURI + "/foo/bar/policy.rego"
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	params, err := ls.fixRenameParams("fix my file!", fileURI)
	if err != nil {
		t.Fatalf("failed to fix rename params: %s", err)
	}

	if params.Label != "fix my file!" {
		t.Fatalf("expected label to be 'Fix my file!', got %s", params.Label)
	}

	change, ok := params.Edit.DocumentChanges[0].(types.RenameFile)
	if !ok {
		t.Fatalf("expected document change to be a RenameFile, got %T", params.Edit.DocumentChanges[0])
	}

	if change.Kind != "rename" {
		t.Fatalf("expected kind to be 'rename', got %s", change.Kind)
	}

	if change.OldURI != fileURI {
		t.Fatalf("expected old URI to be %s, got %s", fileURI, change.OldURI)
	}

	if change.NewURI != fmt.Sprintf("file://%s/workspace/authz/main/rules/policy.rego", tmpDir) {
		t.Fatalf("expected new URI to be 'file://%s/workspace/authz/main/rules/policy.rego', got %s", tmpDir, change.NewURI)
	}
}

func TestLanguageServerFixRenameParamsWithConflict(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testutil.MustMkdirAll(t, tmpDir, "workspace", "foo", "bar")

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierVSCode
	ls.workspaceRootURI = fmt.Sprintf("file://%s/workspace", tmpDir)
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{
			"idiomatic": {
				"directory-package-mismatch": config.Rule{
					Level: "ignore",
					Extra: map[string]any{
						"exclude-test-suffix": true,
					},
				},
			},
		},
	}

	fileURI := ls.workspaceRootURI + "/foo/bar/policy.rego"
	conflictingFileURI := fmt.Sprintf("file://%s/workspace/authz/main/rules/policy.rego", tmpDir)

	ls.cache.SetFileContents(fileURI, "package authz.main.rules")
	ls.cache.SetFileContents(conflictingFileURI, "package authz.main.rules") // existing content irrelevant here

	params, err := ls.fixRenameParams("fix my file!", fileURI)
	if err != nil {
		t.Fatalf("failed to fix rename params: %s", err)
	}

	if params.Label != "fix my file!" {
		t.Fatalf("expected label to be 'Fix my file!', got %s", params.Label)
	}

	if len(params.Edit.DocumentChanges) != 3 {
		t.Fatalf("expected 3 document change, got %d", len(params.Edit.DocumentChanges))
	}

	// check the rename
	change, ok := params.Edit.DocumentChanges[0].(types.RenameFile)
	if !ok {
		t.Fatalf("expected document change to be a RenameFile, got %T", params.Edit.DocumentChanges[0])
	}

	if change.Kind != "rename" {
		t.Fatalf("expected kind to be 'rename', got %s", change.Kind)
	}

	if change.OldURI != fileURI {
		t.Fatalf("expected old URI to be %s, got %s", fileURI, change.OldURI)
	}

	expectedNewURI := fmt.Sprintf("file://%s/workspace/authz/main/rules/policy_1.rego", tmpDir)

	if change.NewURI != expectedNewURI {
		t.Fatalf("expected new URI to be %s, got %s", expectedNewURI, change.NewURI)
	}

	// check the deletes
	deleteChange1, ok := params.Edit.DocumentChanges[1].(types.DeleteFile)
	if !ok {
		t.Fatalf("expected document change to be a DeleteFile, got %T", params.Edit.DocumentChanges[1])
	}

	expectedDeletedURI1 := fmt.Sprintf("file://%s/workspace/foo/bar", tmpDir)
	if deleteChange1.URI != expectedDeletedURI1 {
		t.Fatalf("expected delete URI to be %s, got %s", expectedDeletedURI1, deleteChange1.URI)
	}

	deleteChange2, ok := params.Edit.DocumentChanges[2].(types.DeleteFile)
	if !ok {
		t.Fatalf("expected document change to be a DeleteFile, got %T", params.Edit.DocumentChanges[2])
	}

	expectedDeletedURI2 := fmt.Sprintf("file://%s/workspace/foo", tmpDir)
	if deleteChange2.URI != expectedDeletedURI2 {
		t.Fatalf("expected delete URI to be %s, got %s", expectedDeletedURI2, deleteChange2.URI)
	}
}

func TestLanguageServerFixRenameParamsWhenTargetOutsideRoot(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testutil.MustMkdirAll(t, tmpDir, "workspace", "foo", "bar")

	// this will have regal find a root in the parent dir, which means the file
	// is moved relative to a dir above the workspace.
	testutil.MustWriteFile(t, filepath.Join(tmpDir, ".regal.yaml"), []byte{})

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierVSCode
	// the root where the client stated the workspace is
	// this is what would be set if a config file were in the parent instead
	ls.workspaceRootURI = fmt.Sprintf("file://%s/workspace/", tmpDir)
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{
			"idiomatic": {
				"directory-package-mismatch": config.Rule{
					Level: "ignore",
					Extra: map[string]any{
						"exclude-test-suffix": true,
					},
				},
			},
		},
	}

	fileURI := ls.workspaceRootURI + "foo/bar/policy.rego"
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	if _, err := ls.fixRenameParams("fix my file!", fileURI); err == nil {
		t.Fatalf("expected error, got nil")
	} else if !strings.Contains(err.Error(), "cannot move file out of workspace root") {
		t.Fatalf("expected error to contain 'cannot move file out of workspace root', got %s", err)
	}
}
