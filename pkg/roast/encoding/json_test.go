package encoding

import (
	"bytes"
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
			name: "boolean true",
			json: []byte(`true`),
			want: ast.Boolean(true),
		},
		{
			name: "boolean false",
			json: []byte(`false`),
			want: ast.Boolean(false),
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
	}
)

// Simple routine check to see that things are working as expected.
// While it would be good to add tests for the encoding of each AST node individually, this is thoroughly tested
// via Regal as it consumes the Roast format extensively.

func TestJsonLocationEncoding(t *testing.T) {
	t.Parallel()

	module, err := ast.ParseModuleWithOpts("p.rego", `
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
	`, ast.ParserOptions{ProcessAnnotation: true})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = JSON().Marshal(module); err != nil {
		t.Fatal(err)
	}
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

	mv := OfValue()

	decoders := []struct {
		name string
		fn   func([]byte) (ast.Value, error)
	}{
		{name: "regal", fn: mv.Decode}, {name: "opa", fn: opaDecodeToValue},
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
	decoder := OfValue()

	for _, test := range valueTests {
		b.Run(test.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = decoder.Decode(test.json)
			}
		})

		// uncomment below to compare with OPA's JSON decoder
		// but skip this normally as it's very slow to run all the time

		//nolint:gocritic
		// b.Run(test.name+" OPA decode", func(b *testing.B) {
		// 	for b.Loop() {
		// 		opaDecodeToValue(test.json)
		// 	}
		// })
	}
}

func TestEncodeValue(t *testing.T) {
	t.Parallel()

	decoder := OfValue()

	for _, test := range valueTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			if err := decoder.Encode(buf, test.want); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := must.Return(decoder.Decode(buf.Bytes()))(t)
			if !ast.ValueEqual(test.want, got) {
				t.Fatalf("expected:\n%v\n got:\n%v", test.want, got)
			}
		})
	}
}

func BenchmarkEncodeValue(b *testing.B) {
	decoder := OfValue()
	buf := new(bytes.Buffer)

	for _, test := range valueTests {
		b.Run(test.name, func(b *testing.B) {
			for b.Loop() {
				buf.Reset()
				_ = decoder.Encode(buf, test.want)
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
