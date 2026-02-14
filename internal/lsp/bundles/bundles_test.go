package bundles

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestLoadDataBundle(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		path         string
		files        map[string]string
		expectedData any
	}{
		"simple bundle": {
			path: "foo",
			files: map[string]string{
				"foo/.manifest": `{"roots":["foo"]}`,
				"foo/data.json": `{"foo": "bar"}`,
			},
			expectedData: map[string]any{"foo": "bar"},
		},
		"nested bundle": {
			path: "foo",
			files: map[string]string{
				filepath.FromSlash("foo/.manifest"):     `{"roots":["foo", "bar"]}`,
				filepath.FromSlash("foo/data.yml"):      `foo: bar`,
				filepath.FromSlash("foo/bar/data.yaml"): `bar: baz`,
			},
			expectedData: map[string]any{
				"foo": "bar",
				"bar": map[string]any{"bar": "baz"},
			},
		},
		"array data": {
			path: "foo",
			files: map[string]string{
				filepath.FromSlash("foo/.manifest"):     `{"roots":["bar"]}`,
				filepath.FromSlash("foo/bar/data.json"): `[{"foo": "bar"}]`,
			},
			expectedData: map[string]any{"bar": []any{map[string]any{"foo": "bar"}}},
		},
		"rego files": {
			path: "foo",
			files: map[string]string{
				"foo/.manifest":  `{"roots":["foo"]}`,
				"food/rego.rego": `package foo`,
			},
			expectedData: map[string]any{},
		},
	}

	for testCase, testData := range testCases {
		t.Run(testCase, func(t *testing.T) {
			t.Parallel()

			workspacePath := testutil.TempDirectoryOf(t, testData.files)
			b := must.Return(LoadDataBundle(filepath.Join(workspacePath, testData.path)))(t)

			assert.DeepEqual(t, testData.expectedData, b.Data, "bundle data")
			assert.Equal(t, 0, len(b.Modules), "number of modules")
		})
	}
}
