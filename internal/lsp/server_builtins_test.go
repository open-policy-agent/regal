package lsp

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/config"
)

// https://github.com/open-policy-agent/regal/issues/679
func TestProcessBuiltinUpdateExitsOnMissingFile(t *testing.T) {
	t.Parallel()

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})
	ls.loadedConfig = &config.Config{}

	must.Equal(t, nil, ls.processHoverContentUpdate(t.Context(), "file://missing.rego"))
	assert.Equal(t, 0, len(ls.cache.GetAllBuiltInPositions()), "builtin positions cached")

	contents, ok := ls.cache.GetFileContents("file://missing.rego")
	assert.False(t, ok, "file contents")
	assert.Equal(t, "", contents, "file contents")
	assert.Equal(t, 0, len(ls.cache.GetAllFiles()), "cached files")
}
