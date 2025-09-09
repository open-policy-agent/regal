package capabilities

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestLookupFromFile(t *testing.T) {
	t.Parallel()

	// TODO: https://github.com/open-policy-agent/regal/issues/1683
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// Test that we are able to load a capabilities file using a file:// URL.
	caps, err := Lookup(t.Context(), "file://"+testutil.Must(filepath.Abs("./testdata/capabilities.json"))(t))
	if err != nil {
		t.Errorf("unexpected error from Lookup: %v", err)
	}

	if len(caps.Builtins) != 1 {
		t.Errorf("expected capabilities to have exactly 1 builtin")
	}

	if caps.Builtins[0].Name != "unittest123" {
		t.Errorf("builtin name is incorrect, expected 'unittest123' but got '%s'", caps.Builtins[0].Name)
	}
}

func TestLookupFromEmbedded(t *testing.T) {
	t.Parallel()

	// existing OPA capabilities files from embedded database.
	caps := testutil.Must(Lookup(t.Context(), "regal:///capabilities/opa/v0.55.0"))(t)
	if len(caps.Builtins) != 193 {
		t.Errorf("OPA v0.55.0 capabilities should have 193 builtins, not %d", len(caps.Builtins))
	}
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
		// Note that this actually sorts the input in-place, which is
		// fine since we won't re-visit the same test case twice.
		semverSort(c.input)

		for j, x := range c.expect {
			if x != c.input[j] {
				t.Errorf("index=%d actual='%s' expected='%s'", j, c.input[j], x)
			}
		}
	}
}
