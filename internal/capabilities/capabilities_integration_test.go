//go:build integration

package capabilities

import (
	"context"
	"testing"
)

// Since this test requires internet access, we hide it behind a flag.

func TestLookupFromURL(t *testing.T) {
	t.Parallel()

	// Test that we can load a one of the existing OPA capabilities files
	// via GitHub.

	caps, err := Lookup(
		context.Background(),
		"https://raw.githubusercontent.com/open-policy-agent/opa/main/capabilities/v0.55.0.json",
	)
	if err != nil {
		t.Errorf("unexpected error from Lookup: %v", err)
	}

	if len(caps.Builtins) != 193 {
		t.Errorf("OPA v0.55.0 capabilities should have 193 builtins, not %d", len(caps.Builtins))
	}
}
