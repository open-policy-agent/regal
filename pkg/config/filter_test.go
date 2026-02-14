package config

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestFilterIgnoredPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		paths           []string
		ignore          []string
		checkFileExists bool
		rootDir         string
		expected        []string
	}{
		"no paths": {
			paths:    []string{},
			ignore:   []string{},
			expected: []string{},
		},
		"no ignore": {
			paths:    []string{"foo/bar.rego"},
			ignore:   []string{},
			expected: []string{"foo/bar.rego"},
		},
		"explicit ignore": {
			paths:    []string{"foo/bar.rego", "foo/baz.rego"},
			ignore:   []string{"foo/bar.rego"},
			expected: []string{"foo/baz.rego"},
		},
		"wildcard ignore": {
			paths:    []string{"foo/bar.rego", "foo/baz.rego", "bar/foo.rego"},
			ignore:   []string{"foo/*"},
			expected: []string{"bar/foo.rego"},
		},
		"wildcard ignore, with ext": {
			paths:    []string{"foo/bar.rego", "foo/baz.rego", "bar/foo.rego"},
			ignore:   []string{"foo/*.rego"},
			expected: []string{"bar/foo.rego"},
		},
		"double wildcard ignore": {
			paths:    []string{"foo/bar/baz/bax.rego", "foo/baz/bar/bax.rego", "bar/foo.rego"},
			ignore:   []string{"foo/bar/**"},
			expected: []string{"bar/foo.rego", "foo/baz/bar/bax.rego"},
		},
		"rootDir, explicit ignore": {
			paths:    []string{"wow/foo/bar.rego", "wow/foo/baz.rego"},
			ignore:   []string{"foo/bar.rego"},
			expected: []string{"wow/foo/baz.rego"},
			rootDir:  "wow/",
		},
		"rootDir, no slash, explicit ignore": {
			paths:    []string{"wow/foo/bar.rego", "wow/foo/baz.rego"},
			ignore:   []string{"foo/bar.rego"},
			expected: []string{"wow/foo/baz.rego"},
			rootDir:  "wow",
		},
		"rootDir, wildcard ignore, with ext": {
			paths:    []string{"wow/foo/bar.rego", "wow/foo/baz.rego", "wow/bar/foo.rego"},
			ignore:   []string{"foo/*.rego"},
			expected: []string{"wow/bar/foo.rego"},
			rootDir:  "wow/",
		},
		"rootDir, double wildcard ignore": {
			paths: []string{
				"wow/foo/bar/baz/bax.rego",
				"wow/foo/baz/bar/bax.rego",
				"wow/bar/foo.rego",
			},
			ignore:   []string{"foo/bar/**"},
			expected: []string{"wow/bar/foo.rego", "wow/foo/baz/bar/bax.rego"},
			rootDir:  "wow",
		},
		"rootDir URI": {
			paths: []string{
				"file:///wow/foo/bar.rego",
				"file:///wow/foo/baz.rego",
				"file:///wow/bar/foo.rego",
			},
			ignore:   []string{"foo/*.rego"},
			expected: []string{"file:///wow/bar/foo.rego"},
			rootDir:  "file:///wow",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filtered := must.Return(FilterIgnoredPaths(tc.paths, tc.ignore, tc.checkFileExists, tc.rootDir))(t)

			slices.Sort(filtered)
			slices.Sort(tc.expected)

			if !slices.Equal(filtered, tc.expected) {
				t.Errorf("filtered paths mismatch (-want +got):\n%s", cmp.Diff(tc.expected, filtered))
			}
		})
	}
}

// 8680 ns/op	   13816 B/op	     373 allocs/op // original
// 1241 ns/op	    2040 B/op	      54 allocs/op // optimized
func BenchmarkFilterIgnoredPaths(b *testing.B) {
	paths := []string{
		"foo/bar/baz/bax.rego",
		"foo/baz/bar/bax.rego",
		"bar/foo/regopkg/config/filter.go",
		"bar/foo/regopkg/config/filter_test.go",
		"bar/foo/main.rego",
	}
	ignore := []string{"foo/bar/**", "bar/*.rego"}

	for b.Loop() {
		if _, err := FilterIgnoredPaths(paths, ignore, false, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// 472	   2292632 ns/op	 1251673 B/op	   28348 allocs/op
// 704	   1688877 ns/op	  312571 B/op	    2969 allocs/op
func BenchmarkFilterIgnoredPathsBundleDir(b *testing.B) {
	ignore := []string{"foo/bar/**", "bar/*.rego"}

	for b.Loop() {
		must.Return(FilterIgnoredPaths([]string{"../../bundle"}, ignore, true, ""))(b)
	}
}

func BenchmarkFilterIgnoredPathsWorkspace(b *testing.B) {
	ignore := []string{"foo/bar/**", "bar/*.rego"}

	for b.Loop() {
		must.Return(FilterIgnoredPaths([]string{"../.."}, ignore, true, ""))(b)
	}
}
