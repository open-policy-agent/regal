package module

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/test/must"
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
	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{ProcessAnnotation: true})
	must.Marshal(t, must.Return(ToValue(module))(t))
}

func TestModuleToValueTemplateString(t *testing.T) {
	t.Parallel()

	policy := `package p

	r := $"{x}...{input.bar + 1}"`

	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{ProcessAnnotation: true})

	value, err := ToValue(module)
	if err != nil {
		t.Fatalf("failed to convert module to value: %v", err)
	}

	val, err := value.Find(ast.Ref{
		ast.InternedTerm("rules"),
		ast.InternedTerm(0),
		ast.InternedTerm("head"),
		ast.InternedTerm("value"),
		ast.InternedTerm("value"),
	})
	if err != nil {
		t.Fatalf("failed to find 'rules' key in module object: %v", err)
	}

	tsv := must.Be[ast.Object](t, val)

	if parts := tsv.Get(ast.InternedTerm("parts")); parts != nil {
		must.Equal(t, 3, must.Be[*ast.Array](t, parts.Value).Len())
	} else {
		t.Fatal("expected parts key in template string value")
	}
}

func TestModuleToValueNotImport(t *testing.T) {
	t.Parallel()

	module := ast.MustParseModule(`package test
		import future.keywords.not

		p if {
			not input.denied
		}

		q if {
			not {
				x := input.role
				x == "banned"
			}
		}
	`)
	must.Marshal(t, must.Return(ToValue(module))(t))
}

func TestModuleToValueLogicalKeywords(t *testing.T) {
	t.Parallel()

	module := ast.MustParseModuleWithOpts(`package test
import future.keywords.and
import future.keywords.or

p if {
	input.a or input.b and input.c
}

q if {
	{
		x := input.a
		x == 1
	} and input.b
}
`, ast.ParserOptions{
		Capabilities: ast.CapabilitiesForThisVersion(ast.CapabilitiesExperimentalKeywords(true)),
	})

	value := must.Return(ToValue(module))(t)

	must.Marshal(t, value)

	// `and` binds tighter than `or`, so the first rule is `a or (b and c)`
	or := firstExprTerms(t, value, 0)

	must.Equal(t, "or", stringAttr(t, or, "type"))
	must.Equal(t, 1, bodyAttr(t, or, "lhs").Len())
	must.Equal(t, 1, bodyAttr(t, or, "rhs").Len())

	and := attr(t, bodyAttr(t, or, "rhs").Elem(0).Value, "terms")

	must.Equal(t, "and", stringAttr(t, and, "type"))
	must.Equal(t, 1, bodyAttr(t, and, "lhs").Len())
	must.Equal(t, 1, bodyAttr(t, and, "rhs").Len())

	// the second rule has a brace enclosed lhs, holding two expressions
	explicit := firstExprTerms(t, value, 1)

	must.Equal(t, "and", stringAttr(t, explicit, "type"))
	must.Equal(t, ast.Boolean(true), must.Be[ast.Boolean](t, attr(t, explicit, "explicit_lhs")))
	must.Equal(t, 2, bodyAttr(t, explicit, "lhs").Len())
	must.Equal(t, 1, bodyAttr(t, explicit, "rhs").Len())
}

// firstExprTerms returns the "terms" attribute of the first expression in the
// body of the rule at index i.
func firstExprTerms(t *testing.T, value ast.Value, i int) ast.Value {
	t.Helper()

	return must.Return(value.Find(ast.Ref{
		ast.InternedTerm("rules"),
		ast.InternedTerm(i),
		ast.InternedTerm("body"),
		ast.InternedTerm(0),
		ast.InternedTerm("terms"),
	}))(t)
}

func attr(t *testing.T, value ast.Value, key string) ast.Value {
	t.Helper()

	return must.Be[ast.Object](t, value).Get(ast.InternedTerm(key)).Value
}

func stringAttr(t *testing.T, value ast.Value, key string) string {
	t.Helper()

	return string(must.Be[ast.String](t, attr(t, value, key)))
}

func bodyAttr(t *testing.T, value ast.Value, key string) *ast.Array {
	t.Helper()

	return must.Be[*ast.Array](t, attr(t, value, key))
}

// BenchmarkModuleToValue/ToValue-16         34318 ns/op	   64553 B/op	    1541 allocs/op
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
	module := ast.MustParseModuleWithOpts(policy, ast.ParserOptions{ProcessAnnotation: true})

	var v1, v2 ast.Value

	b.Run("ToValue", func(b *testing.B) {
		for b.Loop() {
			v1 = must.Return(ToValue(module))(b)
		}
	})

	encoding.MustJSONRoundTrip(module, &v2, encoding.NoASTOptions)
	must.Equal(b, 0, v1.Compare(v2), "expected value to equal round-tripped value, got: %v\n\n, want: %v", v1, v2)
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
			obj := ast.NewObjectWithCapacity(n)
			for term := range ast.InternedIntRange(0, n) {
				obj.Insert(term, term)
			}

			if obj.Len() != n {
				b.Errorf("expected object length %d, got %d", n, obj.Len())
			}
		}
	})

	b.Run("New", func(b *testing.B) {
		for b.Loop() {
			terms := make([][2]*ast.Term, 0, n)
			for term := range ast.InternedIntRange(0, n) {
				terms = append(terms, ast.Item(term, term))
			}

			if obj := ast.NewObject(terms...); obj.Len() != n {
				b.Errorf("expected object length %d, got %d", n, obj.Len())
			}
		}
	})
}
