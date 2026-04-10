package util

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

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

// 3.191 ns/op	       0 B/op	       0 allocs/op
// 3912 ns/op	       0 B/op	       0 allocs/op
// 7827 ns/op	       0 B/op	       0 allocs/op
func BenchmarkIndexByteNth(b *testing.B) {
	s := strings.Repeat(strings.Repeat("a", 120)+"\n", 1000)

	for _, n := range []uint{1, 500, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_ = IndexByteNth(s, '\n', n)
			}
		})
	}
}

// 1.86 ns/op	       0 B/op	       0 allocs/op
// 3513 ns/op	       0 B/op	       0 allocs/op
// 7068 ns/op	       0 B/op	       0 allocs/op
func BenchmarkLine(b *testing.B) {
	text := strings.Repeat("this is a line\n", 1000)

	for _, lineNum := range []uint{1, 500, 1000} {
		b.Run(fmt.Sprintf("line %d", lineNum), func(b *testing.B) {
			for b.Loop() {
				_, _ = Line(text, lineNum)
			}
		})
	}
}

// 8462 ns/op	   16384 B/op	       1 allocs/op
// 8145 ns/op	   16384 B/op	       1 allocs/op
// 8173 ns/op	   16384 B/op	       1 allocs/op
func BenchmarkLineBySplit(b *testing.B) {
	text := strings.Repeat("this is a line\n", 1000)

	for _, lineNum := range []uint{1, 500, 1000} {
		b.Run(fmt.Sprintf("line %d", lineNum), func(b *testing.B) {
			for b.Loop() {
				_ = strings.Split(text, "\n")[lineNum-1]
			}
		})
	}
}
