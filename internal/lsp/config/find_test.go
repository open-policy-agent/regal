package config

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/util/test"

	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/util"
)

func TestFindConfigRoots(t *testing.T) {
	t.Parallel()

	fromSlash := util.Mapper(filepath.FromSlash)
	testCases := map[string]struct {
		FS       map[string]string
		Expected []string
	}{
		"no config roots": {
			FS:       map[string]string{},
			Expected: []string{},
		},
		"single config root at root": {
			FS:       map[string]string{".regal/config.yaml": "{}"},
			Expected: fromSlash("/"),
		},
		"single config root at root with .regal.yaml": {
			FS:       map[string]string{".regal.yaml": "{}"},
			Expected: fromSlash("/"),
		},
		"two config roots, one higher": {
			FS:       map[string]string{".regal/config.yaml": "{}", "foo/.regal.yaml": "{}"},
			Expected: fromSlash("/", "/foo"),
		},
		"two config roots, one higher, not in root dir": {
			FS:       map[string]string{"foo/.regal.yaml": "{}", "bar/baz/.regal/config.yaml": "{}"},
			Expected: fromSlash("/bar/baz", "/foo"),
		},
		"two config roots, equal depth": {
			FS:       map[string]string{"bar/.regal/config.yaml": "{}", "foo/.regal.yaml": "{}"},
			Expected: fromSlash("/bar", "/foo"),
		},
	}

	for testName, testData := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			root := test.TempDir(t, testData.FS)
			gotTrimmed := util.Map(must.Return(FindConfigRoots(root))(t), func(path string) string {
				return cmp.Or(strings.TrimPrefix(path, root), filepath.FromSlash("/"))
			})

			if !slices.Equal(gotTrimmed, testData.Expected) {
				t.Fatalf("Expected %v, got %v", testData.Expected, gotTrimmed)
			}
		})
	}
}
