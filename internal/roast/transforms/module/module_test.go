package module

import (
	"strconv"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/transforms"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

func TestModuleToValue(t *testing.T) {
	t.Parallel()

	policy := `# METADATA
# title: p p p
package p

import rego.v1
import data.foo.bar
import data.baz as b

allow := true

allow if some x, y in input

allow if every x in input {
	x > y
}

deny contains "foo"

deny contains "bar" if {
	x == input[_].bar
}

o := {"foo": 1, "quux": {"corge": "grault"}}

x := 1
y := 2.2

arrcomp := [x | some x in input]
objcomp := {x: y | some x, y in input}
setcomp := {x | some x in input}
`
	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
		ProcessAnnotation: true,
	})

	value, err := ToValue(module)
	if err != nil {
		t.Fatalf("failed to convert module to value: %v", err)
	}

	roundTripped, err := roundTripToValue(module)
	if err != nil {
		t.Fatalf("failed to round trip module: %v", err)
	}

	if value.Compare(roundTripped) != 0 {
		t.Errorf("expected value to equal round-tripped value, got: %v\n\n, want: %v", value, roundTripped)
	}
}

// BenchmarkModuleToValue/ToValue-16         	   25171	     46355 ns/op	   65489 B/op	    1802 allocs/op
// BenchmarkModuleToValue/RoundTrip-16       	   10000	    119018 ns/op	  166038 B/op	    3924 allocs/op
func BenchmarkModuleToValue(b *testing.B) {
	policy := `# METADATA
# title: p p p
package p

import rego.v1
import data.foo.bar
import data.baz as b

allow := true

allow if some x, y in input

allow if every x in input {
	x > y
}

deny contains "foo"

deny contains "bar" if {
	x == input[_].bar
}

o := {"foo": 1, "quux": {"corge": "grault"}}

x := 1
y := 2.2

arrcomp := [x | some x in input]
objcomp := {x: y | some x, y in input}
setcomp := {x | some x in input}
`
	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{
		ProcessAnnotation: true,
	})

	var (
		value1, value2 ast.Value
		err            error
	)

	b.Run("ToValue", func(b *testing.B) {
		for b.Loop() {
			value1, err = ToValue(module)
			if err != nil {
				b.Fatalf("failed to convert module to value: %v", err)
			}
		}
	})

	b.Run("RoundTrip", func(b *testing.B) {
		for b.Loop() {
			value2, err = roundTripToValue(module)
			if err != nil {
				b.Fatalf("failed to round trip module: %v", err)
			}
		}
	})

	if value1.Compare(value2) != 0 {
		b.Errorf("expected value to equal round-tripped value, got: %v\n\n, want: %v", value1, value2)
	}
}

func roundTripToValue(module *ast.Module) (ast.Value, error) {
	var obj map[string]any

	encoding.MustJSONRoundTrip(module, &obj)

	return transforms.AnyToValue(obj)
}

// Tangentially related benchmark to find out the cost of repeatedly inserting items into an object
// vs. creating a new object with all items at once. This cost turns out to be insignificant enough
// that I don't think it's worth batching inserts for object creation.
//
// BenchmarkObjectInsertManyVsObjectNew/InsertMany-12         162812      7380 ns/op    9280 B/op     120 allocs/op
// BenchmarkObjectInsertManyVsObjectNew/New-12                210448      5677 ns/op    7544 B/op     107 allocs/op
func BenchmarkObjectInsertManyVsObjectNew(b *testing.B) {
	n := 100 // Number of items to insert

	b.Run("InsertMany", func(b *testing.B) {
		for b.Loop() {
			obj := ast.NewObject()
			for j := range n {
				obj.Insert(ast.InternedTerm(strconv.Itoa(j)), ast.InternedTerm(strconv.Itoa(j)))
			}

			if obj.Len() != n {
				b.Errorf("expected object length %d, got %d", n, obj.Len())
			}
		}
	})

	b.Run("New", func(b *testing.B) {
		for b.Loop() {
			terms := make([][2]*ast.Term, 0, n)
			for j := range n {
				terms = append(terms, ast.Item(ast.InternedTerm(strconv.Itoa(j)), ast.InternedTerm(strconv.Itoa(j))))
			}

			if obj := ast.NewObject(terms...); obj.Len() != n {
				b.Errorf("expected object length %d, got %d", n, obj.Len())
			}
		}
	})
}
