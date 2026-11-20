package templater_test

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/templater"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
	"github.com/open-policy-agent/regal/pkg/rules"
)

func TestTemplateContentsForFile(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		FileKey               string
		CacheFileContents     string
		DiskContents          map[string]string
		RequireConfig         bool
		ServerAllRegoVersions *concurrent.Map[string, ast.RegoVersion]
		ExpectedContents      string
		ExpectedError         string
	}{
		"existing contents in file": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "package foo",
			ExpectedError:     "file already has contents",
		},
		"existing contents on disk": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar.rego": "package foo",
			},
			ExpectedError: "file on disk already has contents",
		},
		"empty file is templated based on root": {
			FileKey:           "foo/bar.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar.rego":       "",
				".regal/config.yaml": "",
			},
			ExpectedContents: "package foo\n\n",
		},
		"empty test file is templated based on root": {
			FileKey:           "foo/bar_test.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar_test.rego":  "",
				".regal/config.yaml": "",
			},
			RequireConfig:    true,
			ExpectedContents: "package foo_test\n\n",
		},
		"empty deeply nested file is templated based on root": {
			FileKey:           "foo/bar/baz/bax.rego",
			CacheFileContents: "",
			DiskContents: map[string]string{
				"foo/bar/baz/bax.rego": "",
				".regal/config.yaml":   "",
			},
			ExpectedContents: "package foo.bar.baz\n\n",
		},
		"v0 templating using rego version setting": {
			FileKey:           filepath.FromSlash("foo/bar/baz/bax.rego"),
			CacheFileContents: "",
			ServerAllRegoVersions: concurrent.MapOf(map[string]ast.RegoVersion{
				"foo": ast.RegoV0,
			}),
			DiskContents: map[string]string{
				filepath.FromSlash("foo/bar/baz/bax.rego"): "",
				filepath.FromSlash(".regal/config.yaml"):   "", // we manually set the versions, config not loaded in these tests
			},
			ExpectedContents: "package foo.bar.baz\n\nimport rego.v1\n",
		},
		"v1 templating using rego version setting": {
			FileKey:           filepath.FromSlash("foo/bar/baz/bax.rego"),
			CacheFileContents: "",
			ServerAllRegoVersions: concurrent.MapOf(map[string]ast.RegoVersion{
				"foo": ast.RegoV1,
			}),
			DiskContents: map[string]string{
				filepath.FromSlash("foo/bar/baz/bax.rego"): "",
				filepath.FromSlash(".regal/config.yaml"):   "", // we manually set the versions, config not loaded in these tests
			},
			ExpectedContents: "package foo.bar.baz\n\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ca := cache.NewCache()
			td := testutil.TempDirectoryOf(t, tc.DiskContents)
			ws := workspace.New(uri.FromPath(clients.IdentifierGeneric, td))
			tp := templater.New(test.DebugLogger(t), ws, ca).
				WithRegoVersionForURI(versionForURI(ws, tc.ServerAllRegoVersions))

			fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, tc.FileKey))

			ca.SetFileContents(fileURI, tc.CacheFileContents)

			newContents, err := tp.Contents(fileURI)
			if tc.ExpectedError != "" {
				testutil.ErrMustContain(err, tc.ExpectedError)(t)
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if newContents != tc.ExpectedContents {
				t.Fatalf("expected contents to be\n%s\ngot\n%s", tc.ExpectedContents, newContents)
			}
		})
	}
}

func TestTemplateContentsForFileInWorkspaceRoot(t *testing.T) {
	t.Parallel()

	td := testutil.TempDirectoryOf(t, map[string]string{".regal/config.yaml": ""})
	cache := cache.NewCache()
	wsRootURI := uri.FromPath(clients.IdentifierGeneric, td)
	workspace := workspace.New(wsRootURI)

	tp := templater.New(test.DebugLogger(t), workspace, cache).WithRegoVersionForURI(versionForURI(workspace, nil))

	fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, "foo.rego"))

	cache.SetFileContents(fileURI, "")
	_, err := tp.Contents(fileURI)

	testutil.ErrMustContain(err, "this function does not template files in the workspace root")(t)
}

func TestTemplateContentsForFileWithUnknownRoot(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	ch := cache.NewCache()
	ws := workspace.New(uri.FromPath(clients.IdentifierGeneric, td))
	tp := templater.New(test.DebugLogger(t), ws, ch).WithRegoVersionForURI(versionForURI(ws, nil))

	fileURI := uri.FromPath(clients.IdentifierGeneric, filepath.Join(td, "foo", "bar.rego"))
	ch.SetFileContents(fileURI, "")

	must.MkdirAll(t, td, "foo")
	assert.Equal(t, "package foo\n\n", must.Return(tp.Contents(fileURI))(t))
}

func versionForURI(
	ws workspace.Workspace,
	cm *concurrent.Map[string, ast.RegoVersion],
) func(fileURI string) ast.RegoVersion {
	return func(fileURI string) ast.RegoVersion {
		if cm != nil {
			return rules.RegoVersionFromMap(cm.Clone(), "/"+ws.RelativePath(fileURI), ast.RegoUndefined)
		}

		return ast.RegoUndefined
	}
}
