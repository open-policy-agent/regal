package rast_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

func TestStructToValue(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2,omitempty"`
		Field3 bool   `json:"field3"`
	}

	input := testStruct{
		Field1: "value1",
		Field2: 0, // This should be omitted due to omitempty
		Field3: true,
	}

	expected := ast.NewObject(
		ast.Item(ast.InternedTerm("field1"), ast.StringTerm("value1")),
		ast.Item(ast.InternedTerm("field3"), ast.BooleanTerm(true)),
	)

	result := rast.StructToValue(input)

	if result.Compare(expected) != 0 {
		t.Errorf("Expected %v, got %v", expected, result)
	}
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

	i := 42
	input := testStruct{
		Field1: "value1",
		Field2: nestedStruct{NestedField: &i},
	}

	expected := ast.NewObject(
		ast.Item(ast.InternedTerm("field1"), ast.StringTerm("value1")),
		ast.Item(ast.InternedTerm("field2"), ast.ObjectTerm(
			ast.Item(ast.InternedTerm("nested_field"), ast.InternedTerm(42)),
		)),
	)

	result := rast.StructToValue(input)

	if result.Compare(expected) != 0 {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// ast.ParseBody-12               114549    9125 ns/op    9604 B/op      96 allocs/op
// ast.ParseRef-12                158643    7653 ns/op    7528 B/op      62 allocs/op
// RefStringToBody-12            4431938     269 ns/op     400 B/op      15 allocs/op
// RefStringToRef-12             5975870     201 ns/op     248 B/op      11 allocs/op
// RefStringToBody_interning-12  7036562     169 ns/op     200 B/op       5 allocs/op
// RefStringToRef_interning-12  11741419     103 ns/op      48 B/op       1 allocs/op
func BenchmarkRefStringToBody(b *testing.B) {
	str := "data.foo.bar.baz.qux.quux"
	ref := ast.NewTerm(ast.MustParseRef(str))
	exp := ast.NewBody(ast.NewExpr(ref))

	b.Run("ast.ParseBody", func(b *testing.B) {
		for b.Loop() {
			if body := ast.MustParseBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("ast.ParseRef", func(b *testing.B) {
		for b.Loop() {
			if r := ast.MustParseRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})

	b.Run("RefStringToBody", func(b *testing.B) {
		for b.Loop() {
			if body := rast.RefStringToBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("RefStringToRef", func(b *testing.B) {
		for b.Loop() {
			if r := rast.RefStringToRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})

	ast.InternStringTerm("foo", "bar", "baz", "qux", "quux")

	b.Run("RefStringToBody_interning", func(b *testing.B) {
		for b.Loop() {
			if body := rast.RefStringToBody(str); !body.Equal(exp) {
				b.Fatalf("expected %v, got %v", exp, body)
			}
		}
	})

	b.Run("RefStringToRef_interning", func(b *testing.B) {
		for b.Loop() {
			if r := rast.RefStringToRef(str); !r.Equal(ref.Value) {
				b.Fatalf("expected %v, got %v", ref.Value, r)
			}
		}
	})
}

func TestRefStringToBody(t *testing.T) {
	t.Parallel()

	tests := []string{"data", "data.foo", "data.foo.bar", "input", "input.foo", "var.string1", "var.string1.string2"}

	for _, test := range tests {
		body := rast.RefStringToBody(test)
		if !body.Equal(ast.MustParseBody(test)) {
			t.Fatalf("expected body to equal %s, got %s", test, body)
		}
	}
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
