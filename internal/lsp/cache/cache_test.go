package cache

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func TestManageAggregates(t *testing.T) {
	t.Parallel()

	reportAggregatesFile1 := ast.NewObject(
		rast.Item("my-rule-category/my-rule-name", ast.SetTerm(
			aggregateObject(map[string]any{"foo": "bar"}),
			aggregateObject(map[string]any{"more": "things"}),
		)),
	)

	reportAggregatesFile2 := ast.NewObject(
		rast.Item("my-rule-category/my-rule-name", ast.SetTerm(
			aggregateObject(map[string]any{"foo": "baz"}),
		)),
		rast.Item("my-other-rule-category/my-other-rule-name", ast.SetTerm(
			aggregateObject(map[string]any{"foo": "bax"}),
		)),
	)

	c := NewCache()
	c.SetFileAggregates("file1.rego", reportAggregatesFile1)
	c.SetFileAggregates("file2.rego", reportAggregatesFile2)

	aggs := c.GetFileAggregates()
	must.Equal(t, 2, len(aggs.Keys()), "number of files with aggregates")

	aggs1, ok := rast.GetValue[ast.Object](aggs, "file1.rego")
	assert.True(t, ok)
	assert.Equal(t, 1, len(aggs1.Keys()), "number of aggregates for file1.rego")

	aggs2, ok := rast.GetValue[ast.Object](aggs, "file2.rego")
	assert.True(t, ok)
	assert.Equal(t, 2, len(aggs2.Keys()), "number of aggregates for file2.rego")

	// update aggregates to only contain file1.rego's aggregates
	c.aggregateData.Delete("file2.rego")

	aggs = c.GetFileAggregates()
	must.Equal(t, 1, len(aggs.Keys()), "numer of aggregates")
	must.NotEqual(t, nil, aggs.Get(ast.InternedTerm("file1.rego")))

	// remove file1 from the cache
	c.Delete("file1.rego")

	must.Equal(t, 0, len(c.GetFileAggregates().Keys()), "number of aggregates")
}

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

func aggregateObject(data map[string]any) *ast.Term {
	o := ast.NewObject(rast.Item("aggregate_source", ast.ObjectTerm(
		rast.Item("package_path", ast.ArrayTerm(ast.InternedTerm("p"))),
	)))

	if len(data) > 0 {
		aggDataObj := ast.NewObject()

		for k, v := range data {
			val, _ := ast.InterfaceToValue(v)
			aggDataObj.Insert(ast.InternedTerm(k), ast.NewTerm(val))
		}

		o.Insert(ast.InternedTerm("aggregate_data"), ast.NewTerm(aggDataObj))
	}

	return ast.NewTerm(o)
}
