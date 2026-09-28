package bundles

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/open-policy-agent/opa/v1/bundle"

	rio "github.com/open-policy-agent/regal/internal/io"
)

var manifestOrData = []string{".manifest", "data.json", "data.yml", "data.yaml"}

// LoadDataBundle loads a bundle from the given path but only includes data
// files. The path must contain a bundle manifest file.
func LoadDataBundle(path string) (bundle.Bundle, error) {
	if !rio.IsFile(filepath.Join(path, ".manifest")) {
		return bundle.Bundle{}, fmt.Errorf("manifest file was not found at bundle path %q", path)
	}

	b, err := bundle.NewCustomReader(bundle.NewDirectoryLoader(path).WithFilter(dataFileLoaderFilter)).Read()
	if err != nil {
		return bundle.Bundle{}, fmt.Errorf("failed to read bundle: %w", err)
	}

	return b, nil
}

func dataFileLoaderFilter(abspath string, info os.FileInfo, _ int) bool {
	return !info.IsDir() && !slices.Contains(manifestOrData, filepath.Base(abspath))
}
