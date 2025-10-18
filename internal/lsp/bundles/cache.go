package bundles

import (
	"bytes"
	//nolint:gosec
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/util"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	rutil "github.com/open-policy-agent/regal/internal/util"
)

// Cache is a struct that maintains a number of bundles in memory and
// provides a way to refresh them when the source files change.
type Cache struct {
	log           *log.Logger
	bundles       map[string]*cacheBundle
	workspacePath string
}

func NewCache(workspacePath string, logger *log.Logger) *Cache {
	return &Cache{
		workspacePath: rutil.EnsureSuffix(workspacePath, string(os.PathSeparator)),
		bundles:       make(map[string]*cacheBundle),
		log:           logger,
	}
}

// cacheBundle is an internal struct that holds a bundle.Bundle and the MD5
// hash of each source file in the bundle. Hashes are used to determine if
// the bundle should be reloaded.
type cacheBundle struct {
	sourceDigests map[string][]byte
	bundle        bundle.Bundle
}

// Refresh walks the workspace path and loads or refreshes any bundles that
// have changed since the last refresh.
func (c *Cache) Refresh() ([]string, error) {
	if c == nil {
		return nil, nil // TODO: is this really needed?
	}

	if c.workspacePath == "" {
		return nil, errors.New("workspace path is empty")
	}

	// find all the bundle roots that are currently present on disk
	foundBundleRoots, err := rio.FindManifestLocations(c.workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to walk workspace path: %w", err)
	}

	var refreshedBundles []string

	// refresh any bundles that have changed
	for _, root := range foundBundleRoots {
		if _, ok := c.bundles[root]; !ok {
			c.bundles[root] = &cacheBundle{}
		}

		refreshed, err := c.bundles[root].Refresh(filepath.Join(c.workspacePath, root))
		if err != nil {
			c.log.Message("failed to refresh bundle %q: %v\n", root, err)

			continue
		}

		if refreshed {
			refreshedBundles = append(refreshedBundles, root)
		}
	}

	// remove any bundles that are no longer present on disk
	maps.DeleteFunc(c.bundles, func(root string, _ *cacheBundle) bool {
		return !slices.Contains(foundBundleRoots, root)
	})

	return refreshedBundles, nil
}

// List returns a list of all the bundle roots that are currently present in
// the cache.
func (c *Cache) List() []string {
	return util.Keys(c.bundles)
}

// Get returns the bundle for the given root from the cache.
func (c *Cache) Get(root string) (bundle.Bundle, bool) {
	b, ok := c.bundles[root]
	if !ok {
		return bundle.Bundle{}, false
	}

	return b.bundle, true
}

// All returns all the bundles in the cache.
func (c *Cache) All() map[string]bundle.Bundle {
	bundles := make(map[string]bundle.Bundle, len(c.bundles))

	for root, cacheBundle := range c.bundles {
		bundles[root] = cacheBundle.bundle
	}

	return bundles
}

// Refresh loads the bundle from disk and updates the cache if any of the
// source files have changed since the last refresh.
func (c *cacheBundle) Refresh(path string) (bool, error) {
	// walk the bundle path and calculate the current MD5 hash of each file on disk
	onDiskDigests, err := files.DefaultWalkReducer(path, make(map[string][]byte)).
		WithFilters(filter.Not(filter.Filenames(".manifest", "data.json", "data.yml", "data.yaml"))).
		Reduce(hasher)
	if err != nil {
		return false, fmt.Errorf("failed to walk bundle path %q: %w", path, err)
	}

	// compare the files on disk with the files that have been seen before
	// and return without reloading the bundle if there have been no changes
	if maps.EqualFunc(c.sourceDigests, onDiskDigests, bytes.Equal) {
		return false, nil
	}

	// update the bundle's sourceDigests to the new on-disk state after a
	// successful refresh
	c.sourceDigests = onDiskDigests

	// if there has been any change in any of the source files, then
	// reload the bundle
	if c.bundle, err = LoadDataBundle(path); err != nil {
		return false, fmt.Errorf("failed to load bundle %q: %w", path, err)
	}

	return true, nil
}

func hasher(path string, curr map[string][]byte) (map[string][]byte, error) {
	return rio.WithOpen(path, func(file *os.File) (map[string][]byte, error) {
		hash := md5.New() //nolint:gosec
		if _, err := io.Copy(hash, file); err != nil {
			return nil, fmt.Errorf("failed to calculate MD5 hash for file %q: %w", path, err)
		}

		curr[path] = hash.Sum(nil)

		return curr, nil
	})
}
