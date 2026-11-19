package config

import (
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
)

// FindConfigRoots will search for all config roots in the given path. A config
// root is a directory that contains a .regal.yaml file or a .regal/config.yaml
// file. This is intended to be used to by the language server when determining
// the most suitable config root for the server to operate on.
func FindConfigRoots(path string) (foundRoots []string, _ error) {
	return files.NewWalker(path).WithFilter(filter.Config).Reduce(foundRoots, configRootsReducer)
}

// HasConfigSuffix reports whether the path is that of a config file, .regal.yaml or .regal/config.yaml.
func HasConfigSuffix(path string) bool {
	return util.HasAnySuffix(path, ".regal.yaml", ".regal/config.yaml", ".regal\\config.yaml")
}

func configRootsReducer(path string, foundRoots []string) ([]string, error) {
	// Normalize path separators to forward slashes for consistent suffix matching
	normalizedPath := filepath.ToSlash(path)

	if strings.HasSuffix(normalizedPath, ".regal.yaml") {
		foundRoots = append(foundRoots, filepath.Dir(path))
	}

	if strings.HasSuffix(normalizedPath, ".regal/config.yaml") {
		foundRoots = append(foundRoots, filepath.Dir(filepath.Dir(path)))
	}

	return foundRoots, nil
}
