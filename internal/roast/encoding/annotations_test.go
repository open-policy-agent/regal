package encoding

import (
	"embed"
	"net/url"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

//go:embed testdata
var testData embed.FS

func TestAnnotationsEncoding(t *testing.T) {
	t.Parallel()

	annotations := ast.Annotations{
		Scope:         "document",
		Title:         "this is a title",
		Entrypoint:    true,
		Description:   "this is a description",
		Organizations: []string{"org1", "org2"},
		RelatedResources: []*ast.RelatedResourceAnnotation{{
			Description: "documentation",
			Ref:         *must.Return(url.Parse("https://example.com"))(t),
		}, {
			Description: "other",
			Ref:         *must.Return(url.Parse("https://example.com/other"))(t),
		}},
		Authors: []*ast.AuthorAnnotation{{
			Name:  "John Doe",
			Email: "john@example.com",
		}, {
			Name:  "Jane Doe",
			Email: "jane@example.com",
		}},
		Schemas: []*ast.SchemaAnnotation{{
			Path:   ast.MustParseRef("input"),
			Schema: ast.MustParseRef("schema.input"),
		}, {
			Path:   ast.MustParseRef("data.foo.bar"),
			Schema: ast.MustParseRef("schema.foo.bar"),
		}, {
			Path:       ast.MustParseRef("data.foo.baz"),
			Definition: new(any(map[string]any{"type": "boolean"})),
		}},
		Custom: map[string]any{
			"key":    "value",
			"object": map[string]any{"nested": "value"},
			"list":   []any{"value1", 2, true},
		},
		Location: &ast.Location{Row: 1, Col: 2, File: "file.rego"},
	}

	roast := must.Return(jsoniter.ConfigFastest.MarshalIndent(annotations, "", "  "))(t)
	expected := mustReadTestFile(t, "testdata/annotations_all.json")
	resultMap := must.Unmarshal[map[string]any](t, roast)
	expectedMap := must.Unmarshal[map[string]any](t, expected)

	// can't compare strings as roast (via jsoniter) does not guarantee order of keys
	assert.DeepEqual(t, expectedMap, resultMap)
}

func mustReadTestFile(tb testing.TB, path string) []byte {
	tb.Helper()

	b, err := testData.ReadFile(path)
	if err != nil {
		tb.Fatalf("Read file %s: %v", path, err)
	}

	return b
}
