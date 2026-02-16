package embedded

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestEmbeddedEOPA(t *testing.T) {
	t.Parallel()

	// Ensure >= 47 embedded capabilities, and that they unmarshal to *ast.Capabilities
	versions := must.Return(LoadCapabilitiesVersions("eopa"))(t)
	assert.True(t, len(versions) >= 47, "minimum number of built-ins")

	for _, v := range versions {
		caps := must.Return(LoadCapabilitiesVersion("eopa", v))(t)
		assert.True(t, len(caps.Builtins) > 0, "eopa capabilities contains built-ins", v)
	}
}

func TestEmbeddedRq(t *testing.T) {
	t.Parallel()

	// As of 2026-02-15, there should be at least 8 rq capabilities files.

	versions, err := LoadCapabilitiesVersions("rq")
	if err != nil {
		t.Fatal(err)
	}

	if len(versions) < 8 {
		t.Errorf("Expected at least 8 rq capabilities in the embedded database (got %d)", len(versions))
	}

	for _, v := range versions {
		caps, err := LoadCapabilitiesVersion("rq", v)
		if err != nil {
			t.Errorf("error with rq capabilities version %s: %v", v, err)
		}

		if len(caps.Builtins) < 1 {
			t.Errorf("rq capabilities version %s has no builtins", v)
		}
	}
}
