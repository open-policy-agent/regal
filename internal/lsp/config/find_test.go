package config

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/util/test"
)

func TestFindConfigRoots(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		FS       map[string]string
		Expected []string
	}{
		"no config roots": {
			FS:       map[string]string{},
			Expected: []string{},
		},
		"single config root at root": {
			FS: map[string]string{
				".regal/config.yaml": "",
			},
			Expected: []string{"/"},
		},
		"single config root at root with .regal.yaml": {
			FS: map[string]string{
				".regal.yaml": "",
			},
			Expected: []string{"/"},
		},
		"two config roots, one higher": {
			FS: map[string]string{
				".regal/config.yaml": "",
				"foo/.regal.yaml":    "",
			},
			Expected: []string{
				filepath.FromSlash("/"),
				filepath.FromSlash("/foo"),
			},
		},
		"two config roots, one higher, not in root dir": {
			FS: map[string]string{
				filepath.FromSlash("foo/.regal.yaml"):            "",
				filepath.FromSlash("bar/baz/.regal/config.yaml"): "",
			},
			Expected: []string{
				filepath.FromSlash("/bar/baz"),
				filepath.FromSlash("/foo"),
			},
		},
		"two config roots, equal depth": {
			FS: map[string]string{
				filepath.FromSlash("bar/.regal/config.yaml"): "",
				filepath.FromSlash("foo/.regal.yaml"):        "",
			},
			Expected: []string{
				filepath.FromSlash("/bar"),
				filepath.FromSlash("/foo"),
			},
		},
	}

	for testName, testData := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			test.WithTempFS(testData.FS, func(root string) {
				got, err := FindConfigRoots(root)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				gotTrimmed := make([]string, len(got))

				for i, path := range got {
					// Normalize path separators to ensure TrimPrefix works correctly on Windows
					normalizedPath := filepath.ToSlash(path)
					normalizedRoot := filepath.ToSlash(root)
					trimmed := cmp.Or(strings.TrimPrefix(normalizedPath, normalizedRoot), "/")
					gotTrimmed[i] = trimmed
				}

				if !slices.Equal(gotTrimmed, testData.Expected) {
					t.Fatalf("Expected %v, got %v", testData.Expected, gotTrimmed)
				}
			})
		})
	}
}
