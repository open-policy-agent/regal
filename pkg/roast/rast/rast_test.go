package rast_test

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func TestStructToValue(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2,omitempty"`
		Field3 bool   `json:"field3"`
	}

	// Field2 should be omitted due to omitempty
	got := rast.StructToValue(testStruct{Field1: "value1", Field2: 0, Field3: true})
	exp := ast.NewObject(rast.Item("field1", ast.InternedTerm("value1")), rast.Item("field3", ast.InternedTerm(true)))

	assert.Equal(t, 0, got.Compare(exp))
}

func TestStructToValueWithRawJSONMessage(t *testing.T) {
	t.Parallel()

	got := rast.StructToValue(struct {
		ID  string          `json:"id"`
		Raw *jsontext.Value `json:"raw"`
	}{
		ID:  "test",
		Raw: new(jsontext.Value(`{"key": "value"}`)),
	})

	exp := ast.NewObject(
		rast.Item("id", ast.InternedTerm("test")),
		rast.Item("raw", ast.ObjectTerm(rast.Item("key", ast.InternedTerm("value")))),
	)

	assert.Equal(t, 0, got.Compare(exp))
}

func TestStructToValueNested(t *testing.T) {
	t.Parallel()

	type nestedStruct struct {
		NestedField *int `json:"nested_field"`
	}

	type testStruct struct {
		Field1 string       `json:"field1"`
		Field2 nestedStruct `json:"field2"`
	}

	got := rast.StructToValue(testStruct{Field1: "value1", Field2: nestedStruct{NestedField: new(42)}})
	exp := ast.NewObject(
		rast.Item("field1", ast.InternedTerm("value1")),
		rast.Item("field2", ast.ObjectTerm(rast.Item("nested_field", ast.InternedTerm(42)))),
	)

	assert.Equal(t, 0, got.Compare(exp))
}

func TestStructToValueWithASTValueField(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Field ast.Value `json:"field"`
	}

	got := rast.StructToValue(testStruct{Field: ast.NewObject(rast.Item("key", ast.InternedTerm("value")))})
	exp := ast.NewObject(rast.Item("field", ast.ObjectTerm(rast.Item("key", ast.InternedTerm("value")))))

	assert.Equal(t, 0, got.Compare(exp))
}

func TestStructToValueTagWithTrailingComma(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Field string `json:"field,"` //nolint
	}

	got := rast.StructToValue(testStruct{Field: "value"})
	exp := ast.NewObject(rast.Item("field", ast.InternedTerm("value")))

	assert.Equal(t, 0, got.Compare(exp))
}

// BenchmarkAppendLocation/single_line_no_prealloc-16         34704147        34.05 ns/op       8 B/op       1 allocs/op
// BenchmarkAppendLocation/multi_line_no_prealloc-16          29631702        39.94 ns/op      16 B/op       1 allocs/op
// BenchmarkAppendLocation/single_line_with_prealloc-16       41071040        27.80 ns/op       0 B/op       0 allocs/op
// BenchmarkAppendLocation/multi_line_with_prealloc-16        30247112        40.32 ns/op       0 B/op       0 allocs/op
func BenchmarkAppendLocation(b *testing.B) {
	cases := []struct {
		name     string
		location *ast.Location
		prealloc []byte
	}{{
		name:     "single line no prealloc",
		location: &ast.Location{Row: 3, Col: 5, Text: []byte("example text")},
	}, {
		name:     "multi line no prealloc",
		location: &ast.Location{Row: 2, Col: 10, Text: []byte("line one\nline two\nline three")},
	}, {
		name:     "single line with prealloc",
		location: &ast.Location{Row: 1, Col: 1, Text: []byte("single line")},
		prealloc: make([]byte, 0, 10),
	}, {
		name:     "multi line with prealloc",
		location: &ast.Location{Row: 4, Col: 3, Text: []byte("first line\nsecond line\nthird line\nfourth line")},
		prealloc: make([]byte, 0, 20),
	}}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = rast.AppendLocation(tc.prealloc, tc.location)
			}
		})
	}
}
