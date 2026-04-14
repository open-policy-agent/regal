package cache

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestPartialDiagnosticsUpdate(t *testing.T) {
	t.Parallel()

	diag1 := types.Diagnostic{Code: "code1"}
	diag2 := types.Diagnostic{Code: "code2"}
	diag3 := types.Diagnostic{Code: "code3"}

	c := NewCache()
	c.SetFileDiagnostics("foo.rego", []types.Diagnostic{diag1, diag2})

	foundDiags := testutil.MustBeOK(c.GetFileDiagnostics("foo.rego"))(t)
	assert.DeepEqual(t, []types.Diagnostic{diag1, diag2}, foundDiags, "diagnostics")

	c.SetFileDiagnosticsForRules("foo.rego", []string{"code2", "code3"}, []types.Diagnostic{diag3})

	foundDiags = testutil.MustBeOK(c.GetFileDiagnostics("foo.rego"))(t)
	assert.DeepEqual(t, []types.Diagnostic{diag1, diag3}, foundDiags, "diagnostics")
}

func TestCacheRename(t *testing.T) {
	t.Parallel()

	c := NewCache()
	c.SetFileContents("file:///tmp/foo.rego", "package foo")
	c.SetModule("file:///tmp/foo.rego", &ast.Module{})
	c.Rename("file:///tmp/foo.rego", "file:///tmp/bar.rego")

	assert.False(t, c.HasFileContents("file:///tmp/foo.rego"))

	contents := testutil.MustBeOK(c.GetFileContents("file:///tmp/bar.rego"))(t)
	must.Equal(t, "package foo", contents, "file contents")
}
