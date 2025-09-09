package encoding

import (
	"embed"
	"net/url"
	"reflect"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
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
		RelatedResources: []*ast.RelatedResourceAnnotation{
			{
				Description: "documentation",
				Ref:         MustParseURL(t, "https://example.com"),
			},
			{
				Description: "other",
				Ref:         MustParseURL(t, "https://example.com/other"),
			},
		},
		Authors: []*ast.AuthorAnnotation{
			{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			{
				Name:  "Jane Doe",
				Email: "jane@example.com",
			},
		},
		Schemas: []*ast.SchemaAnnotation{
			{
				Path:   ast.MustParseRef("input"),
				Schema: ast.MustParseRef("schema.input"),
			},
			{
				Path:   ast.MustParseRef("data.foo.bar"),
				Schema: ast.MustParseRef("schema.foo.bar"),
			},
			{
				Path: ast.MustParseRef("data.foo.baz"),
				Definition: mapToAnyPointer(map[string]any{
					"type": "boolean",
				}),
			},
		},
		Custom: map[string]any{
			"key": "value",
			"object": map[string]any{
				"nested": "value",
			},
			"list": []any{"value1", 2, true},
		},
		Location: &ast.Location{
			Row:  1,
			Col:  2,
			File: "file.rego",
		},
	}

	// Test encoding
	json := jsoniter.ConfigFastest

	roast, err := json.MarshalIndent(annotations, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal annotations: %v", err)
	}

	if !json.Valid(roast) {
		if err = json.Unmarshal(roast, &map[string]any{}); err != nil {
			t.Fatalf("produced invalid JSON: %v", err)
		}
	}

	var resultMap map[string]any

	if err = json.Unmarshal(roast, &resultMap); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expected := mustReadTestFile(t, "testdata/annotations_all.json")

	var expectedMap map[string]any

	err = json.Unmarshal(expected, &expectedMap)
	if err != nil {
		t.Fatalf("failed to unmarshal expected JSON: %v", err)
	}

	// can't compare string representation as roast (via jsoniter) does
	// not guarantee order of keys
	if !reflect.DeepEqual(expectedMap, resultMap) {
		t.Fatalf("expected %s, got %s", expected, roast)
	}
}

func MustParseURL(t *testing.T, s string) url.URL {
	t.Helper()

	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	return *u
}

func mapToAnyPointer(m map[string]any) *any {
	var p any = m

	return &p
}

func mustReadTestFile(tb testing.TB, path string) []byte {
	tb.Helper()

	b, err := testData.ReadFile(path)
	if err != nil {
		tb.Fatalf("Read file %s: %v", path, err)
	}

	return b
}
