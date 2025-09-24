package completions

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/completions/providers"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestManagerEarlyExitInsideComment(t *testing.T) {
	t.Parallel()

	fileURI := "file:///foo/bar/file.rego"
	fileContents := "package p\n\n# foo := http\n"
	module := ast.MustParseModule(fileContents)

	c := cache.NewCache()
	c.SetFileContents(fileURI, fileContents)
	c.SetModule(fileURI, module)

	m := map[string]any{"workspace": map[string]any{"parsed": map[string]any{fileURI: module}}}
	store := inmem.NewFromObjectWithOpts(m, inmem.OptRoundTripOnWrite(false))
	opts := &providers.Options{}
	mgr := NewDefaultManager(t.Context(), c, store, query.NewCache())

	completions := testutil.Must(mgr.Run(t.Context(), types.NewCompletionParams(fileURI, 2, 13, nil), opts))(t)
	if len(completions) != 0 {
		t.Errorf("Expected no completions, got: %v", completions)
	}
}
