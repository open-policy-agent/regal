package fileprovider

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestFromFS(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDirectoryOf(t, map[string]string{
		filepath.FromSlash("foo/bar/baz"): "bar",
		filepath.FromSlash("bar/foo"):     "baz",
	})
	fp := must.Return(NewInMemoryFileProviderFromFS(
		filepath.Join(tempDir, "foo", "bar", "baz"),
		filepath.Join(tempDir, "bar", "foo"),
	))(t)

	if fc, err := fp.Get(filepath.Join(tempDir, "foo", "bar", "baz")); err != nil || fc != "bar" {
		t.Fatalf("expected %s, got %s", "bar", fc)
	}
}

func TestRenameConflict(t *testing.T) {
	t.Parallel()

	fp := NewInMemoryFileProvider(map[string]string{
		filepath.FromSlash("/foo/bar/baz"): "bar",
		filepath.FromSlash("/bar/foo"):     "baz",
	})
	exp := fmt.Sprintf("rename conflict: %q cannot be renamed as the target location %q already exists",
		filepath.FromSlash("/foo/bar/baz"), filepath.FromSlash("/bar/foo"))

	testutil.ErrMustContain(fp.Rename(filepath.FromSlash("/foo/bar/baz"), filepath.FromSlash("/bar/foo")), exp)(t)
}
