package fileprovider

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestCacheFileProvider(t *testing.T) {
	t.Parallel()

	c := cache.NewCache()
	c.SetFileContents("file:///tmp/foo.rego", "package foo")
	c.SetFileContents("file:///tmp/bar.rego", "package bar")

	cfp := NewCacheFileProvider(c, clients.IdentifierGeneric)

	assert.Equal(t, nil, cfp.Put("file:///tmp/foo.rego", "package wow"), "put file")
	assert.Equal(t, "package wow", must.Return(cfp.Get("file:///tmp/foo.rego"))(t), "cache updated")
	assert.Equal(t, nil, cfp.Rename("file:///tmp/foo.rego", "file:///tmp/wow.rego"), "rename file")
	assert.True(t, cfp.deletedFiles.Contains("file:///tmp/foo.rego"), "file deleted")
	assert.True(t, cfp.modifiedFiles.Contains("file:///tmp/wow.rego"), "file modified")
	assert.Equal(t, "package wow", must.Return(cfp.Get("file:///tmp/wow.rego"))(t), "file contents")
}
