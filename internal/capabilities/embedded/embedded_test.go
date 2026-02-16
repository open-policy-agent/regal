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
