package bundles

import (
	"io"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestRefresh(t *testing.T) {
	t.Parallel()

	workspacePath := testutil.TempDirectoryOf(t, map[string]string{
		"foo/.manifest": `{"roots":["foo"]}`,
		"foo/data.json": `{"foo": "bar"}`,
	})

	c := NewCache(workspacePath, log.NewLogger(log.LevelOff, io.Discard))

	// perform the first load of the bundles
	refreshedBundles := testutil.Must(c.Refresh())(t)

	if !slices.Equal(refreshedBundles, []string{"foo"}) {
		t.Fatalf("unexpected refreshed bundles: %v", refreshedBundles)
	}

	if len(c.List()) != 1 {
		t.Fatalf("unexpected number of bundles: %d", len(c.List()))
	}

	fooBundle := testutil.MustBeOK(c.Get("foo"))(t)
	if !maps.Equal(fooBundle.Data, map[string]any{"foo": "bar"}) {
		t.Fatalf("unexpected bundle data: %v", fooBundle.Data)
	}

	if fooBundle.Manifest.Roots == nil {
		t.Fatalf("unexpected bundle roots: %v", fooBundle.Manifest.Roots)
	}

	if !slices.Equal(*fooBundle.Manifest.Roots, []string{"foo"}) {
		t.Fatalf("unexpected bundle roots: %v", *fooBundle.Manifest.Roots)
	}

	// perform the second load of the bundles, after no changes on disk
	refreshedBundles = testutil.Must(c.Refresh())(t)

	if !slices.Equal(refreshedBundles, []string{}) {
		t.Fatalf("unexpected refreshed bundles: %v", refreshedBundles)
	}

	// add a new unrelated file
	testutil.MustWriteFile(t, filepath.Join(workspacePath, "foo", "foo.rego"), []byte(`package wow`))

	// perform the third load of the bundles, after adding a new unrelated file
	refreshedBundles = testutil.Must(c.Refresh())(t)
	if !slices.Equal(refreshedBundles, []string{}) {
		t.Fatalf("unexpected refreshed bundles: %v", refreshedBundles)
	}

	// update the data in the bundle
	testutil.MustWriteFile(t, filepath.Join(workspacePath, "foo", "data.json"), []byte(`{"foo": "baz"}`))

	refreshedBundles = testutil.Must(c.Refresh())(t)
	if !slices.Equal(refreshedBundles, []string{"foo"}) {
		t.Fatalf("unexpected refreshed bundles: %v", refreshedBundles)
	}

	fooBundle = testutil.MustBeOK(c.Get("foo"))(t)
	if !maps.Equal(fooBundle.Data, map[string]any{"foo": "baz"}) {
		t.Fatalf("unexpected bundle data: %v", fooBundle.Data)
	}

	// create a new bundle
	testutil.MustMkdirAll(t, workspacePath, "bar")
	testutil.MustWriteFile(t, filepath.Join(workspacePath, "bar", ".manifest"), []byte(`{"roots":["bar"]}`))
	testutil.MustWriteFile(t, filepath.Join(workspacePath, "bar", "data.json"), []byte(`{"bar": true}`))

	refreshedBundles = testutil.Must(c.Refresh())(t)
	if !slices.Equal(refreshedBundles, []string{"bar"}) {
		t.Fatalf("unexpected refreshed bundles: %v", refreshedBundles)
	}

	barBundle := testutil.MustBeOK(c.Get("bar"))(t)
	if !maps.Equal(barBundle.Data, map[string]any{"bar": true}) {
		t.Fatalf("unexpected bundle data: %v", fooBundle.Data)
	}

	// remove the foo bundle
	testutil.MustRemoveAll(t, workspacePath, "foo")

	_ = testutil.Must(c.Refresh())(t)
	if !slices.Equal(c.List(), []string{"bar"}) {
		t.Fatalf("unexpected bundle list: %v", c.List())
	}
}
