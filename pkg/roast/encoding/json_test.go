package encoding

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/test/must"
)

var (
	_ = intern("a", "b", "c", "d", "e", "f", "nested", "object", "two", "interned")

	valueTests = []struct {
		name string
		json []byte
		want ast.Value
	}{
		{
			name: "not interned string",
			json: []byte(`"not interned"`),
			want: ast.String("not interned"),
		},
		{
			name: "interned string",
			json: []byte(`"interned"`),
			want: ast.InternedValue("interned"),
		},
		{
			name: "string with escapes",
			json: []byte(`"data.regal.rules.bugs[\"argument-always-wildcard_test\"]"`),
			want: ast.InternedValue(`data.regal.rules.bugs["argument-always-wildcard_test"]`),
		},
		{
			name: "boolean true",
			json: []byte(`true`),
			want: ast.InternedBooleanTrueValue,
		},
		{
			name: "boolean false",
			json: []byte(`false`),
			want: ast.InternedBooleanFalseValue,
		},
		{
			name: "null",
			json: []byte(`null`),
			want: ast.NullValue,
		},
		{
			name: "interned integer",
			json: []byte(`50`),
			want: ast.Number("50"),
		},
		{
			name: "not interned integer",
			json: []byte(`123456789012345678901234567890`),
			want: ast.Number("123456789012345678901234567890"),
		},
		{
			name: "float",
			json: []byte(`3.14`),
			want: ast.Number("3.14"),
		},
		{
			name: "array",
			json: []byte(`[1, "two", true, null, [3, 4]]`),
			want: ast.MustParseTerm(`[1, "two", true, null, [3, 4]]`).Value,
		},
		{
			name: "empty array",
			json: []byte(`[]`),
			want: ast.InternedEmptyArrayValue,
		},
		{
			name: "object",
			json: []byte(`{"a": 1, "b": "two", "c": true, "d": null, "e": {"nested": "object"}}`),
			want: ast.MustParseTerm(
				`{"a": 1, "b": "two", "c": true, "d": null, "e": {"nested": "object"}}`).Value,
		},
		{
			name: "empty object",
			json: []byte(`{}`),
			want: ast.InternedEmptyObjectValue,
		},
		{
			name: "complex object",
			json: []byte(`{"label":"data.regal.rules.bugs[\"argument-always-wildcard_test\"]"}`),
			want: ast.MustParseTerm(`{"label":"data.regal.rules.bugs[\"argument-always-wildcard_test\"]"}`).Value,
		},
	}
)

// Simple routine check to see that things are working as expected.
// While it would be good to add tests for the encoding of each AST node individually, this is thoroughly tested
// via Regal as it consumes the Roast format extensively.

func TestJsonLocationEncoding(t *testing.T) {
	t.Parallel()

	must.Marshal(t, ast.MustParseModuleWithOpts(`
package p

import rego.v1

import data.foo.bar

# METADATA
# description: foo bar went to the bar
allow if true

# regular comment

add(x, y) := x + y

partial[x] contains y if {
	some x, y in input

	every z in x {
		z == y
	}
}

obj := {"foo": {"number": 1}, "string": {"set"}, "bool": false}

arr := [1, {"foo": {"key": 1}}]

sc := {x | x := [1, 2, 3][_]}

ac := [x | x := [1, 2, 3][_]]

oc := {k:v | some k, v in input}

test_foo if {
	allow with input as {"foo": "bar"}
}
	`, ast.ParserOptions{ProcessAnnotation: true}))
}

// https://github.com/open-policy-agent/regal/issues/1592
func TestJSONRoundTripBigNumber(t *testing.T) {
	t.Parallel()

	module := ast.MustParseModule("package p\n\nn := 1e400")

	var modMap map[string]any
	if err := JSONRoundTrip(module, &modMap); err != nil {
		t.Fatalf("failed to marshal module: %v", err)
	}
}

func TestDecodeToValue(t *testing.T) {
	t.Parallel()

	regalDecodeToValue := func(bs []byte) (ast.Value, error) {
		return JSONUnmarshalTo[ast.Value](bs, NoASTOptions)
	}

	decoders := []struct {
		name string
		fn   func([]byte) (ast.Value, error)
	}{
		{name: "regal", fn: regalDecodeToValue},
		{name: "opa", fn: opaDecodeToValue},
	}

	for _, test := range valueTests {
		for _, decoder := range decoders {
			t.Run(test.name+" "+decoder.name, func(t *testing.T) {
				t.Parallel()

				got, err := decoder.fn(test.json)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !ast.ValueEqual(test.want, got) {
					t.Fatalf("expected:\n%v\n got:\n%v", test.want, got)
				}
			})
		}
	}
}

func BenchmarkDecodeToValue(b *testing.B) {
	for _, test := range valueTests {
		b.Run(test.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = JSONUnmarshalTo[ast.Value](test.json, NoASTOptions)
			}
		})

		b.Run(test.name+" OPA decode", func(b *testing.B) {
			for b.Loop() {
				_, _ = opaDecodeToValue(test.json)
			}
		})
	}
}

func TestValueNoASTRoundTrip(t *testing.T) {
	t.Parallel()

	for _, test := range valueTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			exp := must.Marshal(t, test.want, NoASTOptions)
			got := must.UnmarshalTo[ast.Value](t, exp, NoASTOptions)

			if !ast.ValueEqual(test.want, got) {
				t.Fatalf("expected:\n%v\n got:\n%v", test.want, got)
			}
		})
	}
}

func BenchmarkValueNoASTEncode(b *testing.B) {
	for _, test := range valueTests {
		b.Run(test.name, func(b *testing.B) {
			for b.Loop() {
				must.Equal(b, nil, json.MarshalWrite(io.Discard, test.want, NoASTOptions))
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	// Example test cases for the Get function
	tests := []struct {
		name string
		json string
		path jsontext.Pointer
		want jsontext.Value
	}{
		{
			name: "object key",
			json: `{"foo": "bar"}`,
			path: "/foo",
			want: jsontext.Value(`"bar"`),
		},
		{
			name: "nested object key",
			json: `{"foo": {"bar": "baz"}}`,
			path: "/foo/bar",
			want: jsontext.Value(`"baz"`),
		},
		{
			name: "nested object second object",
			json: `{"position":{"character":16,"line":5},"textDocument":{"uri":"file:///foo.txt"}}`,
			path: "/textDocument/uri",
			want: jsontext.Value(`"file:///foo.txt"`),
		},
		{
			name: "non-existent key",
			json: `{"foo": "bar"}`,
			path: "/baz",
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Get(jsontext.Value(test.json), test.path)
			if !bytes.Equal(got, test.want) {
				t.Fatalf("expected:\n%v\n got:\n%v", test.want, got)
			}
		})
	}
}

// hack to ensure this has been called before any tests are set up
func intern(s ...string) bool {
	ast.InternStringTerm(s...)

	return true
}

// as found in OPA's cmd/eval.go
func opaDecodeToValue(bs []byte) (value ast.Value, err error) {
	var input any
	if err = util.Unmarshal(bs, &input); err == nil {
		value, err = ast.InterfaceToValue(input)
	}

	return value, err
}
