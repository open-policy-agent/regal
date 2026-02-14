package bundles

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
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
	refreshedBundles := must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{"foo"}, refreshedBundles, "refresh first load")
	must.Equal(t, 1, len(c.List()), "number of bundles")

	fooBundle := testutil.MustBeOK(c.Get("foo"))(t)
	assert.MapsEqual(t, map[string]any{"foo": "bar"}, fooBundle.Data, "bundle data")
	must.NotEqual(t, nil, fooBundle.Manifest.Roots, "bundle roots")
	assert.SlicesEqual(t, []string{"foo"}, *fooBundle.Manifest.Roots, "bundle roots")

	// perform the second load of the bundles, after no changes on disk
	refreshedBundles = must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{}, refreshedBundles, "refresh no changes")

	// add a new unrelated file
	must.WriteFile(t, filepath.Join(workspacePath, "foo", "foo.rego"), []byte(`package wow`))

	// perform the third load of the bundles, after adding a new unrelated file
	refreshedBundles = must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{}, refreshedBundles, "refresh unrelated file")
	// update the data in the bundle
	must.WriteFile(t, filepath.Join(workspacePath, "foo", "data.json"), []byte(`{"foo": "baz"}`))

	refreshedBundles = must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{"foo"}, refreshedBundles, "refresh updated bundle")

	fooBundle = testutil.MustBeOK(c.Get("foo"))(t)
	assert.MapsEqual(t, map[string]any{"foo": "baz"}, fooBundle.Data, "bundle data")

	// create a new bundle
	must.MkdirAll(t, workspacePath, "bar")
	must.WriteFile(t, filepath.Join(workspacePath, "bar", ".manifest"), []byte(`{"roots":["bar"]}`))
	must.WriteFile(t, filepath.Join(workspacePath, "bar", "data.json"), []byte(`{"bar": true}`))

	refreshedBundles = must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{"bar"}, refreshedBundles, "refresh new bundle")

	barBundle := testutil.MustBeOK(c.Get("bar"))(t)
	assert.MapsEqual(t, map[string]any{"bar": true}, barBundle.Data, "bundle data")

	// remove the foo bundle
	must.RemoveAll(t, workspacePath, "foo")

	_ = must.Return(c.Refresh())(t)
	assert.SlicesEqual(t, []string{"bar"}, c.List(), "bundle list")
}
