package capabilities

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestLookupFromFile(t *testing.T) {
	t.Parallel()

	// TODO: https://github.com/open-policy-agent/regal/issues/1683
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// Test that we are able to load a capabilities file using a file:// URL.
	caps, err := Lookup(t.Context(), "file://"+must.Return(filepath.Abs("./testdata/capabilities.json"))(t))

	assert.Equal(t, nil, err, "unexpected error from Lookup")
	assert.Equal(t, 1, len(caps.Builtins), "expected capabilities to have exactly 1 builtin")
	assert.Equal(t, "unittest123", caps.Builtins[0].Name, "builtin name incorrect")
}

func TestLookupFromEmbedded(t *testing.T) {
	t.Parallel()

	// existing OPA capabilities files from embedded database.
	caps := must.Return(Lookup(t.Context(), "regal:///capabilities/opa/v0.55.0"))(t)
	assert.Equal(t, 193, len(caps.Builtins), "OPA v0.55.0 caps, unexpected number of builtins")
}

func TestSemverSort(t *testing.T) {
	t.Parallel()

	cases := []struct {
		note   string
		input  []string
		expect []string
	}{
		{
			note:   "should be able to correctly sort semver only",
			input:  []string{"1.2.3", "1.2.4", "1.0.1"},
			expect: []string{"1.2.4", "1.2.3", "1.0.1"},
		},
		{
			note:   "should be able to correctly sort non-semver only",
			input:  []string{"a", "b", "c"},
			expect: []string{"c", "b", "a"},
		},
		{
			note:   "should be able to correctly sort mixed semver and non-semver",
			input:  []string{"a", "b", "c", "4.0.7", "1.0.1", "2.1.1", "2.3.4"},
			expect: []string{"4.0.7", "2.3.4", "2.1.1", "1.0.1", "c", "b", "a"},
		},
	}

	for _, c := range cases {
		t.Run(c.note, func(t *testing.T) {
			t.Parallel()

			semverSort(c.input)

			for j, x := range c.expect {
				assert.Equal(t, x, c.input[j], "index=%d", j)
			}
		})
	}
}
