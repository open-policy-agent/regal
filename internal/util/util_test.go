package util

import (
	"slices"
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
		// Windows-style path tests
		{
			name:     "windows different length roots",
			roots:    []string{`C:\a\b\c`, `C:\a\b`, `C:\a`},
			path:     `C:\a\b\c\d\e\f`,
			expected: `C:\a\b\c`,
		},
		{
			name:     "windows exact match",
			roots:    []string{`C:\a\b\c`, `C:\a\b`, `C:\a`},
			path:     `C:\a\b`,
			expected: `C:\a\b`,
		},
		{
			name:     "windows mixed roots",
			roots:    []string{`C:\a\b\c\b\a`, `C:\c\b`, `C:\a\d\c\f`},
			path:     `C:\c\b\a`,
			expected: `C:\c\b`,
		},
		{
			name:     "windows no matching root",
			roots:    []string{`C:\a\b\c`},
			path:     `C:\d`,
			expected: "",
		},
		{
			name:     "windows with drive letters",
			roots:    []string{`D:\root\main`, `C:\root`, `C:\workspace`},
			path:     `C:\root\main\test\file.rego`,
			expected: `C:\root`,
		},
		// Mixed separator tests (shouldn't happen in practice)
		{
			name:     "unix path with windows roots (no match expected)",
			roots:    []string{`C:\a\b`, `C:\a`},
			path:     "/a/b/c",
			expected: "",
		},
		{
			name:     "windows path with unix roots (no match expected)",
			roots:    []string{"/a/b", "/a"},
			path:     `C:\a\b\c`,
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

// No allocations
func BenchmarkSorted(b *testing.B) {
	unsorted := []string{
		"the", "quick", "brown", "fox", "jumped", "over", "the", "lazy", "dog",
		"foo", "bar", "baz", "qux", "quux", "corge", "grault", "garply", "waldo", "fred", "plugh", "xyzzy", "thud",
		"x", "y", "z", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
	}
	sorted := []string{
		"a", "b", "bar", "baz", "brown", "c", "corge", "d", "dog", "e", "f", "foo", "fox", "fred", "g", "garply",
		"grault", "h", "i", "j", "jumped", "lazy", "over", "plugh", "quick", "quux", "qux", "the", "the", "thud",
		"waldo", "x", "xyzzy", "y", "z",
	}

	var got []string
	for b.Loop() {
		got = Sorted(unsorted)
	}

	if !slices.Equal(got, sorted) {
		b.Fatalf("expected %v, got %v", sorted, got)
	}
}
