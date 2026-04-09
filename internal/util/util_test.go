package util

import (
	"fmt"
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

func TestIndexByteNth(t *testing.T) {
	t.Parallel()

	for n, exp := range []int{-1, 3, 7, 11, -1} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()

			must.Equal(t, exp, IndexByteNth("foo\nbar\nbaz\nqux", '\n', uint(n))) //nolint:gosec
		})
	}
}

func TestLine(t *testing.T) {
	t.Parallel()

	text := "line1\nline2\nline3\nline4"

	for i, exp := range []string{"line1", "line2", "line3", "line4"} {
		i++

		t.Run(fmt.Sprintf("line %d", i), func(t *testing.T) {
			t.Parallel()

			line, ok := Line(text, uint(i)) //nolint:gosec
			must.Equal(t, true, ok, "not ok at line %d", i)
			must.Equal(t, exp, line, "content at line %d", i)
		})
	}

	for _, lineNum := range []uint{0, 5} {
		t.Run(fmt.Sprintf("line %d", lineNum), func(t *testing.T) {
			t.Parallel()

			line, ok := Line(text, lineNum)
			must.Equal(t, false, ok)
			must.Equal(t, "", line)
		})
	}
}
