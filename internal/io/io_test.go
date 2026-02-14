package io

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/util/test"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
)

func TestFindManifestLocations(t *testing.T) {
	t.Parallel()

	fs := map[string]string{
		filepath.FromSlash("/.git"):                          "",
		filepath.FromSlash("/foo/bar/baz/.manifest"):         "",
		filepath.FromSlash("/foo/bar/qux/.manifest"):         "",
		filepath.FromSlash("/foo/bar/.regal/.manifest.yaml"): "",
		filepath.FromSlash("/node_modules/.manifest"):        "",
	}

	test.WithTempFS(fs, func(root string) {
		locations, err := FindManifestLocations(root)
		expected := util.Map([]string{"foo/bar/baz", "foo/bar/qux"}, filepath.FromSlash)

		must.Equal(t, nil, err)
		assert.SlicesEqual(t, expected, locations, "manifest locations")
	})
}

func TestDirCleanUpPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		State                     map[string]string
		DeleteTarget              string
		AdditionalPreserveTargets []string
		Expected                  []string
	}{
		"simple": {
			DeleteTarget: "foo/bar.rego",
			State: map[string]string{
				"foo/bar.rego": "package foo",
			},
			Expected: []string{"foo"},
		},
		"not empty": {
			DeleteTarget: "foo/bar.rego",
			State: map[string]string{
				"foo/bar.rego": "package foo",
				"foo/baz.rego": "package foo",
			},
			Expected: []string{},
		},
		"all the way up": {
			DeleteTarget: filepath.FromSlash("foo/bar/baz/bax.rego"),
			State: map[string]string{
				filepath.FromSlash("foo/bar/baz/bax.rego"): "package baz",
			},
			Expected: []string{filepath.FromSlash("foo/bar/baz"), filepath.FromSlash("foo/bar"), "foo"},
		},
		"almost all the way up": {
			DeleteTarget: "foo/bar/baz/bax.rego",
			State: map[string]string{
				"foo/bar/baz/bax.rego": "package baz",
				"foo/bax.rego":         "package foo",
			},
			Expected: []string{"foo/bar/baz", "foo/bar"},
		},
		"with preserve targets": {
			DeleteTarget: "foo/bar/baz/bax.rego",
			AdditionalPreserveTargets: []string{
				"foo/bar/baz_test/bax.rego",
			},
			State: map[string]string{
				"foo/bar/baz/bax.rego": "package baz",
				"foo/bax.rego":         "package foo",
			},
			// foo/bar is not deleted because of the preserve target
			Expected: []string{"foo/bar/baz"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutil.TempDirectoryOf(t, test.State)
			expected := util.Map(test.Expected, util.FilepathJoiner(tempDir))

			additionalPreserveTargets := []string{tempDir}
			for i, v := range test.AdditionalPreserveTargets {
				additionalPreserveTargets[i] = filepath.Join(tempDir, v)
			}

			deleteTarget := filepath.Join(tempDir, test.DeleteTarget)
			got := must.Return(DirCleanUpPaths(deleteTarget, additionalPreserveTargets))(t)

			assert.SlicesEqual(t, expected, got, "clean up paths")
		})
	}
}

func TestCapabilitiesNoDuplicateBuiltins(t *testing.T) {
	t.Parallel()

	builtinSet := util.NewSet[string]()
	for _, b := range Capabilities().Builtins {
		must.Equal(t, false, builtinSet.Contains(b.Name), "duplicate builtin found: %s", b.Name)
		builtinSet.Add(b.Name)
	}
}

func TestCapabilitiesIncludeRegalBuiltins(t *testing.T) {
	t.Parallel()

	expectedBuiltins := util.NewSet("regal.parse_module", "regal.last", "regal.is_formatted")
	found := util.NewSet[string]()

	for _, b := range Capabilities().Builtins {
		if expectedBuiltins.Contains(b.Name) {
			found.Add(b.Name)
		}
	}

	assert.True(t, expectedBuiltins.Equal(found))
}

func TestOPACapabilitiesIncludeNoRegalBuiltins(t *testing.T) {
	t.Parallel()

	for _, b := range OPACapabilities().Builtins {
		must.Equal(t, false, strings.HasPrefix(b.Name, "regal."), "regal builtin in opa capabilities: %s", b.Name)
	}
}

func BenchmarkLoadRegalBundlePath(b *testing.B) {
	for b.Loop() {
		must.Return(LoadRegalBundlePath("../../bundle"))(b)
	}
}
