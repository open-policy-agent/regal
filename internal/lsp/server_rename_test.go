package lsp

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestLanguageServerFixRenameParams(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	must.MkdirAll(t, tmpDir, "workspace", "foo", "bar")

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierGeneric
	ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace"))
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := uri.FromRelativePath(ls.client.Identifier, "foo/bar/policy.rego", ls.workspaceRootURI)
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	params := must.Return(ls.fixRenameParams("fix my file!", fileURI))(t)
	must.Equal(t, "fix my file!", params.Label, "label")

	change := must.Be[types.RenameFile](t, params.Edit.DocumentChanges[0])
	must.Equal(t, "rename", change.Kind, "change kind")
	must.Equal(t, fileURI, change.OldURI, "old URI")

	path := filepath.Join(tmpDir, "workspace", "authz", "main", "rules", "policy.rego")
	must.Equal(t, uri.FromPath(clients.IdentifierGeneric, path), change.NewURI, "new URI")
}

func TestLanguageServerFixRenameParamsWithConflict(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	must.MkdirAll(t, tmpDir, "workspace", "foo", "bar")

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierGeneric
	ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace"))
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := uri.FromRelativePath(ls.client.Identifier, "foo/bar/policy.rego", ls.workspaceRootURI)
	conflictingFileURI := uri.FromPath(
		clients.IdentifierGeneric,
		filepath.Join(tmpDir, "workspace", "authz", "main", "rules", "policy.rego"),
	)

	ls.cache.SetFileContents(fileURI, "package authz.main.rules")
	ls.cache.SetFileContents(conflictingFileURI, "package authz.main.rules") // existing content irrelevant here

	params := must.Return(ls.fixRenameParams("fix my file!", fileURI))(t)
	must.Equal(t, "fix my file!", params.Label, "label")
	must.Equal(t, 3, len(params.Edit.DocumentChanges), "number of document changes")

	// check the rename
	change := must.Be[types.RenameFile](t, params.Edit.DocumentChanges[0])
	must.Equal(t, "rename", change.Kind, "change kind")
	must.Equal(t, fileURI, change.OldURI, "old URI")

	path := filepath.Join(tmpDir, "workspace", "authz", "main", "rules", "policy_1.rego")
	must.Equal(t, uri.FromPath(clients.IdentifierGeneric, path), change.NewURI, "new URI")

	// check the deletes
	deleteChange1 := must.Be[types.DeleteFile](t, params.Edit.DocumentChanges[1])

	expectedDeletedURI1 := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace", "foo", "bar"))
	must.Equal(t, expectedDeletedURI1, deleteChange1.URI, "delete URI")

	deleteChange2 := must.Be[types.DeleteFile](t, params.Edit.DocumentChanges[2])

	expectedDeletedURI2 := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace", "foo"))
	must.Equal(t, expectedDeletedURI2, deleteChange2.URI, "delete URI")
}

func TestLanguageServerFixRenameParamsWhenTargetOutsideRoot(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	must.MkdirAll(t, tmpDir, "workspace", "foo", "bar")

	// this will have regal find a root in the parent dir, which means the file
	// is moved relative to a dir above the workspace.
	must.WriteFile(t, filepath.Join(tmpDir, ".regal.yaml"), []byte{})

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.client.Identifier = clients.IdentifierGeneric
	// the root where the client stated the workspace is
	// this is what would be set if a config file were in the parent instead
	ls.workspaceRootURI = uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace"))
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := uri.FromRelativePath(ls.client.Identifier, "foo/bar/policy.rego", ls.workspaceRootURI)
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	_, err := ls.fixRenameParams("fix my file!", fileURI)
	assert.StringContains(t, err.Error(), "cannot move file out of workspace root")
}
