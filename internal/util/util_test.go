package util

import (
	"slices"
	"strconv"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/must"
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
			name:  "no matching root",
			roots: []string{"/a/b/c"},
			path:  "/d",
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
			name:  "windows no matching root",
			roots: []string{`C:\a\b\c`},
			path:  `C:\d`,
		},
		{
			name:     "windows with drive letters",
			roots:    []string{`D:\root\main`, `C:\root`, `C:\workspace`},
			path:     `C:\root\main\test\file.rego`,
			expected: `C:\root`,
		},
		// Mixed separator tests (shouldn't happen in practice)
		{
			name:  "unix path with windows roots (no match expected)",
			roots: []string{`C:\a\b`, `C:\a`},
			path:  "/a/b/c",
		},
		{
			name:  "windows path with unix roots (no match expected)",
			roots: []string{"/a/b", "/a"},
			path:  `C:\a\b\c`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			must.Equal(t, test.expected, FindClosestMatchingRoot(test.path, test.roots))
		})
	}
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

func TestLineContents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     uint
		expected string
	}{
		{name: "first line", line: 0, expected: "line1"},
		{name: "middle line", line: 1, expected: "line2"},
		{name: "last line", line: 3, expected: "line4"},
		{name: "out of bounds (too high)", line: 5, expected: ""},
	}

	src := []byte("line1\nline2\nline3\nline4")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			must.Equal(t, test.expected, string(LineContents(src, test.line)), "line contents")
		})
	}
}

// 9090 ns/op    24576 B/op    1 allocs/op // return bytes.Split(document, []byte{'\n'})[lineNum]
// 4726 ns/op        0 B/op    0 allocs/op // current implementation
func BenchmarkLineContents(b *testing.B) {
	src := []byte{}
	for i := range uint64(1000) {
		src = append(strconv.AppendUint(append(src, "This is line number "...), i, 10), '\n')
	}

	b.Run("LineContents", func(b *testing.B) {
		for b.Loop() {
			_ = LineContents(src, 500)
		}
	})
}
