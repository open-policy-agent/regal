package embedded_test

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/io/embedded"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func BenchmarkLoadRegalBundlePath(b *testing.B) {
	for b.Loop() {
		must.Return(embedded.FromPath("../../../bundle"))(b)
	}
}
