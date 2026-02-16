package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/util"
)

func TestEvalWorkspacePath(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	ls.workspaceRootURI = "file:///workspace"

	policy1 := `package policy1

	import data.policy2

	default allow := false

	allow if policy2.allow
	`

	policy2 := `package policy2

	allow if {
		print(1)
		input.exists
	}
	`

	policy1URI := uri.FromRelativePath(ls.client.Identifier, "policy1.rego", ls.workspaceRootURI)
	policy1RelativeFileName := uri.ToRelativePath(policy1URI, ls.workspaceRootURI)
	module1 := must.Return(rparse.ModuleWithOpts(policy1RelativeFileName, policy1, rparse.ParserOptions()))(t)

	policy2URI := uri.FromRelativePath(ls.client.Identifier, "policy2.rego", ls.workspaceRootURI)
	policy2RelativeFileName := uri.ToRelativePath(policy2URI, ls.workspaceRootURI)
	module2 := must.Return(rparse.ModuleWithOpts(policy2RelativeFileName, policy2, rparse.ParserOptions()))(t)

	ls.cache.SetFileContents(policy1URI, policy1)
	ls.cache.SetFileContents(policy2URI, policy2)
	ls.cache.SetModule(policy1URI, module1)
	ls.cache.SetModule(policy2URI, module2)

	input := map[string]any{"exists": true}

	res := must.Return(ls.EvalInWorkspace(t.Context(), "data.policy1.allow", input))(t)
	assert.True(t, must.Be[bool](t, res.Value))

	expectedPrintOutput := map[string]map[int][]string{policy2URI: {4: {"1"}}}
	must.Equal(t, "", cmp.Diff(expectedPrintOutput, res.PrintOutput), "print output")
}

func TestEvalWorkspacePathInternalData(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	res := must.Return(ls.EvalInWorkspace(t.Context(), "object.keys(data.internal)", map[string]any{}))(t)
	val := must.Be[[]any](t, res.Value)
	act := util.Sorted(must.Return(util.AnySliceTo[string](val))(t))

	assert.SlicesEqual(t, []string{"capabilities", "combined_config", "user_config"}, act)
}

func TestFindInputPath(t *testing.T) {
	t.Parallel()

	cases := []struct{ fileExt, fileContent string }{{"json", `{"x": true}`}, {"yaml", "x: true"}}

	for _, tc := range cases {
		t.Run(tc.fileExt, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			workspacePath := filepath.Join(tmpDir, "workspace")
			file := filepath.Join(tmpDir, "workspace", "foo", "bar", "baz.rego")

			must.MkdirAll(t, workspacePath, "foo", "bar")
			must.Equal(t, "", rio.FindInputPath(file, workspacePath), "expected no input path to be found")

			inputPath := filepath.Join(workspacePath, "foo", "bar", "input."+tc.fileExt)
			createWithContent(t, inputPath, tc.fileContent)

			assert.Equal(t, inputPath, rio.FindInputPath(file, workspacePath), "input")
			must.Remove(t, inputPath)

			workspaceInputPath := filepath.Join(workspacePath, "input."+tc.fileExt)
			createWithContent(t, workspaceInputPath, tc.fileContent)

			assert.Equal(t, workspaceInputPath, rio.FindInputPath(file, workspacePath), "input")
		})
	}
}

func TestFindInput(t *testing.T) {
	t.Parallel()

	cases := []struct{ fileType, fileContent string }{{"json", `{"x": true}`}, {"yaml", "x: true"}}

	for _, tc := range cases {
		t.Run(tc.fileType, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			workspacePath := filepath.Join(tmpDir, "workspace")
			file := filepath.Join(tmpDir, "workspace", "foo", "bar", "baz.rego")

			must.MkdirAll(t, workspacePath, "foo", "bar")

			path, content := rio.FindInput(file, workspacePath)
			must.Equal(t, "", path, "expected no input path to be found")
			assert.MapsEqual(t, map[string]any{}, content, "expected no input content")

			inputPath := filepath.Join(workspacePath, "foo", "bar", "input."+tc.fileType)

			createWithContent(t, inputPath, tc.fileContent)

			path, content = rio.FindInput(file, workspacePath)
			assert.Equal(t, inputPath, path, "input path")
			assert.MapsEqual(t, map[string]any{"x": true}, content, "input content")

			must.Remove(t, inputPath)

			workspaceInputPath := filepath.Join(workspacePath, "input."+tc.fileType)
			createWithContent(t, workspaceInputPath, tc.fileContent)

			path, content = rio.FindInput(file, workspacePath)
			assert.Equal(t, workspaceInputPath, path, "input path")
			assert.MapsEqual(t, map[string]any{"x": true}, content, "input content")
		})
	}
}

func createWithContent(t *testing.T, path, content string) {
	t.Helper()

	must.Equal(t, nil, rio.WithCreateRecursive(path, func(f *os.File) error {
		_, err := f.WriteString(content)

		return err
	}))
}
