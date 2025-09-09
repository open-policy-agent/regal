package fileprovider

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestCacheFileProvider(t *testing.T) {
	t.Parallel()

	c := cache.NewCache()
	c.SetFileContents("file:///tmp/foo.rego", "package foo")
	c.SetFileContents("file:///tmp/bar.rego", "package bar")

	cfp := NewCacheFileProvider(c, clients.IdentifierGeneric)

	testutil.NoErr(cfp.Put("file:///tmp/foo.rego", "package wow"))(t)

	if contents := testutil.Must(cfp.Get("file:///tmp/foo.rego"))(t); contents != "package wow" {
		t.Fatalf("expected %s, got %s", "package wow", contents)
	}

	if contentsStr := testutil.MustBeOK(c.GetFileContents("file:///tmp/foo.rego"))(t); contentsStr != "package wow" {
		t.Fatalf("expected %s, got %s", "package wow", contentsStr)
	}

	testutil.NoErr(cfp.Rename("file:///tmp/foo.rego", "file:///tmp/wow.rego"))(t)

	if !cfp.deletedFiles.Contains("file:///tmp/foo.rego") {
		t.Fatalf("expected file to be deleted")
	}

	if !cfp.modifiedFiles.Contains("file:///tmp/wow.rego") {
		t.Fatalf("expected file to be modified")
	}

	if contents := testutil.Must(cfp.Get("file:///tmp/wow.rego"))(t); contents != "package wow" {
		t.Fatalf("expected %s, got %s", "package wow", contents)
	}
}
