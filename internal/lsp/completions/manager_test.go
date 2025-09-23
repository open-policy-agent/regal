package completions

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/completions/providers"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

func TestManagerEarlyExitInsideComment(t *testing.T) {
	t.Parallel()

	fileURI := "file:///foo/bar/file.rego"
	fileContents := "package p\n\n# foo := http\n"

	c := cache.NewCache()
	c.SetFileContents(fileURI, fileContents)
	c.SetModule(fileURI, ast.MustParseModule(fileContents))

	mgr := NewManager(c, &ManagerOptions{})
	mgr.RegisterProvider(&providers.PackageRefs{})

	completions, err := mgr.Run(t.Context(), types.NewCompletionParams(fileURI, 2, 13, nil), nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(completions) != 0 {
		t.Errorf("Expected no completions, got: %v", completions)
	}
}
