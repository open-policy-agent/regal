package lsp

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/client"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestLanguageServerFixRenameParams(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	must.MkdirAll(t, tmpDir, "workspace", "foo", "bar")

	wsRootURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace"))
	workspace := workspace.New(wsRootURI).WithClient(client.NewGeneric())

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	ls.workspace = workspace
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := workspace.URI("foo", "bar", "policy.rego")
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	changes := must.Return(ls.fixRenameChanges(fileURI))(t)

	change := must.Be[types.RenameFile](t, changes[0])
	must.Equal(t, "rename", change.Kind, "change kind")
	must.Equal(t, fileURI, change.OldURI, "old URI")

	path := filepath.Join(tmpDir, "workspace", "authz", "main", "rules", "policy.rego")
	must.Equal(t, uri.FromPath(clients.IdentifierGeneric, path), change.NewURI, "new URI")
}

func TestLanguageServerFixRenameParamsWithConflict(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	must.MkdirAll(t, tmpDir, "workspace", "foo", "bar")

	wsRootURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace"))
	workspace := workspace.New(wsRootURI).WithClient(client.NewGeneric())

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	ls.workspace = workspace
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := workspace.URI("foo", "bar", "policy.rego")
	conflictingFileURI := workspace.URI("authz", "main", "rules", "policy.rego")

	ls.cache.SetFileContents(fileURI, "package authz.main.rules")
	ls.cache.SetFileContents(conflictingFileURI, "package authz.main.rules") // existing content irrelevant here

	changes := must.Return(ls.fixRenameChanges(fileURI))(t)
	must.Equal(t, 3, len(changes), "number of document changes")

	// check the rename
	change := must.Be[types.RenameFile](t, changes[0])
	must.Equal(t, "rename", change.Kind, "change kind")
	must.Equal(t, fileURI, change.OldURI, "old URI")

	path := filepath.Join(tmpDir, "workspace", "authz", "main", "rules", "policy_1.rego")
	must.Equal(t, uri.FromPath(clients.IdentifierGeneric, path), change.NewURI, "new URI")

	// check the deletes
	deleteChange1 := must.Be[types.DeleteFile](t, changes[1])

	expectedDeletedURI1 := uri.FromPath(clients.IdentifierGeneric, filepath.Join(tmpDir, "workspace", "foo", "bar"))
	must.Equal(t, expectedDeletedURI1, deleteChange1.URI, "delete URI")

	deleteChange2 := must.Be[types.DeleteFile](t, changes[2])

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

	client := client.NewGeneric()
	wsRootURI := client.URIFromPath(filepath.Join(tmpDir, "workspace"))
	workspace := workspace.New(wsRootURI).WithClient(client)

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: test.DebugLogger(t)})
	must.Equal(t, nil, ls.loadWorkspace(t.Context(), wsRootURI, client))

	// the root where the client stated the workspace is
	// this is what would be set if a config file were in the parent instead
	ls.workspace = workspace
	ls.loadedConfig = &config.Config{
		Rules: map[string]config.Category{"idiomatic": {
			"directory-package-mismatch": config.Rule{
				Level: "ignore",
				Extra: map[string]any{"exclude-test-suffix": true},
			},
		}},
	}

	fileURI := uri.FromRelativePath(clients.IdentifierGeneric, "foo/bar/policy.rego", workspace.URI())
	ls.cache.SetFileContents(fileURI, "package authz.main.rules")

	_, err := ls.fixRenameChanges(fileURI)
	assert.StringContains(t, err.Error(), "cannot move file out of workspace root")
}
