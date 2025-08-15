package util

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
)

func TestFindClosestMatchingRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		roots    []string
		path     string
		expected string
	}{
		{
			name:     "different length roots",
			roots:    []string{"/a/b/c", "/a/b", "/a"},
			path:     "/a/b/c/d/e/f",
			expected: "/a/b/c",
		},
		{
			name:     "exact match",
			roots:    []string{"/a/b/c", "/a/b", "/a"},
			path:     "/a/b",
			expected: "/a/b",
		},
		{
			name:     "mixed roots",
			roots:    []string{"/a/b/c/b/a", "/c/b", "/a/d/c/f"},
			path:     "/c/b/a",
			expected: "/c/b",
		},
		{
			name:     "no matching root",
			roots:    []string{"/a/b/c"},
			path:     "/d",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := FindClosestMatchingRoot(test.path, test.roots)
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func BenchmarkStringRepeatMake(b *testing.B) {
	for b.Loop() {
		_ = stringRepeatMake("test", 1000)
	}
}

func stringRepeatMake(s string, n int) []*ast.Term {
	sl := make([]*ast.Term, n)
	for i := range s {
		sl[i] = &ast.Term{Value: ast.String("test")}
	}

	return sl
}

// Without pre-allocating, this is more than twice as slow and results in 5 allocs/op.
// BenchmarkFilter/Filter-10    5919769    191.0 ns/op    224 B/op    1 allocs/op
// ...
func BenchmarkFilter(b *testing.B) {
	strings := []string{
		"foo", "bar", "baz", "qux", "quux", "corge", "grault", "garply", "waldo", "fred", "plugh", "xyzzy", "thud",
		"x", "y", "z", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "the", "lazy", "dog", "jumped", "over",
		"the", "quick", "brown", "fox",
	}

	pred := func(s string) bool {
		return len(s) > 3
	}

	b.Run("Filter", func(b *testing.B) {
		for b.Loop() {
			_ = Filter(strings, pred)
		}
	})
}
