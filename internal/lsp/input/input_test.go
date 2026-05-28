package input_test

import (
	"testing"
	"testing/fstest"

	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/input"
	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
)

func TestFindForPath(t *testing.T) {
	t.Parallel()

	st := inmem.NewFromObject(map[string]any{"workspace": map[string]any{"inputs": map[string]any{}}})
	im := input.NewManager(st, test.DebugLogger(t))

	inputJSON, inputYAML := []byte(`{"foo": "bar"}`), []byte(`foo: bar`)

	im.LoadFromWorkspace(t.Context(), workspace.New("file:///").WithFS(fstest.MapFS{
		"input.json":             {Data: inputJSON},
		"foo/bar/input.yaml":     {Data: inputYAML},
		"foo/bar/baz/input.json": {Data: inputJSON},
	}))

	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "most specific match", path: "foo/bar/baz/p.rego", want: "foo/bar/baz/input.json"},
		{name: "less specific match", path: "foo/bar/p.rego", want: "foo/bar/input.yaml"},
		{name: "workspace root match", path: "p.rego", want: "input.json"},
		{name: "no file match", path: "foo/baz/p.rego", want: "input.json"},
		{name: "most specific directory match", path: "foo/bar/baz", want: "foo/bar/baz/input.json"},
		{name: "less specific directory match", path: "foo/bar", want: "foo/bar/input.yaml"},
		{name: "workspace root directory match", path: "", want: "input.json"},
		{name: "directory trailing slash match", path: "foo/bar/baz/", want: "foo/bar/baz/input.json"},
		{name: "directory leading slash match", path: "/foo/bar/baz", want: "foo/bar/baz/input.json"},
		{name: "match from uri", path: "file:///foo/bar/baz", want: "foo/bar/baz/input.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if path := im.FindForPath(tc.path); path != tc.want {
				t.Errorf("expected path %q, got %q", tc.want, path)
			}
		})
	}
}
