package cache

import (
	"reflect"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/types"
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
	if len(aggs.Keys()) != 2 { // file1.rego and file2.rego
		t.Fatalf("unexpected number of aggregates: %d", len(aggs.Keys()))
	}

	aggs1 := aggs.Get(ast.InternedTerm("file1.rego")).Value.(ast.Object)
	if len(aggs1.Keys()) != 1 { // there is one cat/rule for file1
		t.Fatalf("unexpected number of aggregates for file1.rego: %d", len(aggs1.Keys()))
	}

	aggs2 := aggs.Get(ast.InternedTerm("file2.rego")).Value.(ast.Object)
	if len(aggs2.Keys()) != 2 {
		t.Fatalf("unexpected number of aggregates for file2.rego: %d", len(aggs2.Keys()))
	}

	// update aggregates to only contain file1.rego's aggregates
	c.aggregateData.Delete("file2.rego")

	aggs = c.GetFileAggregates()
	if len(aggs.Keys()) != 1 { // only file1.rego should remain
		t.Fatalf("unexpected number of aggregates: %d", len(aggs.Keys()))
	}

	if aggs.Get(ast.InternedTerm("file1.rego")) == nil {
		t.Fatalf("expected file1.rego aggregates to remain")
	}

	// remove file1 from the cache
	c.Delete("file1.rego")

	aggs = c.GetFileAggregates()
	if len(aggs.Keys()) != 0 {
		t.Fatalf("unexpected number of aggregates: %d", len(aggs.Keys()))
	}
}

func TestPartialDiagnosticsUpdate(t *testing.T) {
	t.Parallel()

	diag1 := types.Diagnostic{Code: "code1"}
	diag2 := types.Diagnostic{Code: "code2"}
	diag3 := types.Diagnostic{Code: "code3"}

	c := NewCache()
	c.SetFileDiagnostics("foo.rego", []types.Diagnostic{diag1, diag2})

	foundDiags := testutil.MustBeOK(c.GetFileDiagnostics("foo.rego"))(t)
	if !reflect.DeepEqual(foundDiags, []types.Diagnostic{diag1, diag2}) {
		t.Fatalf("unexpected diagnostics: %v", foundDiags)
	}

	c.SetFileDiagnosticsForRules("foo.rego", []string{"code2", "code3"}, []types.Diagnostic{diag3})

	foundDiags = testutil.MustBeOK(c.GetFileDiagnostics("foo.rego"))(t)
	if !reflect.DeepEqual(foundDiags, []types.Diagnostic{diag1, diag3}) {
		t.Fatalf("unexpected diagnostics: %v", foundDiags)
	}
}

func TestCacheRename(t *testing.T) {
	t.Parallel()

	c := NewCache()
	c.SetFileContents("file:///tmp/foo.rego", "package foo")
	c.SetModule("file:///tmp/foo.rego", &ast.Module{})
	c.Rename("file:///tmp/foo.rego", "file:///tmp/bar.rego")

	if ok := c.HasFileContents("file:///tmp/foo.rego"); ok {
		t.Fatalf("expected foo.rego to be removed")
	}

	if contents := testutil.MustBeOK(c.GetFileContents("file:///tmp/bar.rego"))(t); contents != "package foo" {
		t.Fatalf("unexpected contents: %s", contents)
	}
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
