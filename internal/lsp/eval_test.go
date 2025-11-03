package lsp

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
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
	module1 := testutil.Must(rparse.ModuleWithOpts(policy1RelativeFileName, policy1, rparse.ParserOptions()))(t)

	policy2URI := uri.FromRelativePath(ls.client.Identifier, "policy2.rego", ls.workspaceRootURI)
	policy2RelativeFileName := uri.ToRelativePath(policy2URI, ls.workspaceRootURI)
	module2 := testutil.Must(rparse.ModuleWithOpts(policy2RelativeFileName, policy2, rparse.ParserOptions()))(t)

	ls.cache.SetFileContents(policy1URI, policy1)
	ls.cache.SetFileContents(policy2URI, policy2)
	ls.cache.SetModule(policy1URI, module1)
	ls.cache.SetModule(policy2URI, module2)

	input := map[string]any{"exists": true}

	res := testutil.Must(ls.EvalInWorkspace(t.Context(), "data.policy1.allow", input))(t)
	if val, ok := res.Value.(bool); !ok || val != true {
		t.Fatalf("expected true, got false")
	}

	expectedPrintOutput := map[string]map[int][]string{policy2URI: {4: {"1"}}}
	if diff := cmp.Diff(expectedPrintOutput, res.PrintOutput); diff != "" {
		t.Fatalf("unexpected print output (-want +got):\n%s", diff)
	}
}

func TestEvalWorkspacePathInternalData(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})

	res := testutil.Must(ls.EvalInWorkspace(t.Context(), "object.keys(data.internal)", map[string]any{}))(t)
	val := testutil.MustBe[[]any](t, res.Value)
	act := util.Sorted(testutil.Must(util.AnySliceTo[string](val))(t))

	if exp := []string{"capabilities", "combined_config"}; !slices.Equal(exp, act) {
		t.Fatalf("expected %v, got %v", exp, act)
	}
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

			testutil.MustMkdirAll(t, workspacePath, "foo", "bar")

			if path := rio.FindInputPath(file, workspacePath); path != "" {
				t.Fatalf("did not expect to find input.%s", tc.fileExt)
			}

			inputPath := filepath.Join(workspacePath, "foo", "bar", "input."+tc.fileExt)
			createWithContent(t, inputPath, tc.fileContent)

			if path, exp := rio.FindInputPath(file, workspacePath), inputPath; path != exp {
				t.Errorf(`expected input at %s, got %s`, exp, path)
			}

			testutil.MustRemove(t, inputPath)

			workspaceInputPath := filepath.Join(workspacePath, "input."+tc.fileExt)
			createWithContent(t, workspaceInputPath, tc.fileContent)

			if path, exp := rio.FindInputPath(file, workspacePath), workspaceInputPath; path != exp {
				t.Errorf(`expected input at %s, got %s`, exp, path)
			}
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

			testutil.MustMkdirAll(t, workspacePath, "foo", "bar")

			if path, content := rio.FindInput(file, workspacePath); path != "" || content != nil {
				t.Fatalf("did not expect to find input.%s", tc.fileType)
			}

			inputPath := filepath.Join(workspacePath, "foo", "bar", "input."+tc.fileType)

			createWithContent(t, inputPath, tc.fileContent)

			path, content := rio.FindInput(file, workspacePath)
			if path != inputPath {
				t.Errorf(`expected input at %s, got %s`, inputPath, path)
			}

			if !maps.Equal(content, map[string]any{"x": true}) {
				t.Errorf(`expected input {"x": true}, got %s`, content)
			}

			testutil.MustRemove(t, inputPath)

			workspaceInputPath := filepath.Join(workspacePath, "input."+tc.fileType)
			createWithContent(t, workspaceInputPath, tc.fileContent)

			path, content = rio.FindInput(file, workspacePath)
			if path != workspaceInputPath {
				t.Errorf(`expected input at %s, got %s`, workspaceInputPath, path)
			}

			if !maps.Equal(content, map[string]any{"x": true}) {
				t.Errorf(`expected input {"x": true} at, got %s`, content)
			}
		})
	}
}

func createWithContent(t *testing.T, path string, content string) {
	t.Helper()

	testutil.NoErr(rio.WithCreateRecursive(path, func(f *os.File) error {
		_, err := f.WriteString(content)

		return err
	}))(t)
}
