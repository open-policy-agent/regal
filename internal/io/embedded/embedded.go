package embedded

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/loader/filter"

	"github.com/open-policy-agent/regal/internal/io"
)

var (
	excludeTests       filter.LoaderFilter = excludeTestFilter
	excludeTestsAndLSP filter.LoaderFilter = excludeTestAndLanguageServerFilter
)

// LoadFull loads embedded bundle without tests, panics on error.
func LoadFull(fs fs.FS) *bundle.Bundle {
	return mustLoad(fs, excludeTests)
}

// LoadNoLSP loads embedded bundle excluding tests and language server files, panics on error.
func LoadNoLSP(fs fs.FS) *bundle.Bundle {
	return mustLoad(fs, excludeTestsAndLSP)
}

// FromPath loads bundle from path.
func FromPath(path string) (*bundle.Bundle, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open root: %w", err)
	}
	defer io.CloseIgnore(root)

	return loadWithFilter(root.FS(), excludeTests)
}

func loadWithFilter(fs fs.FS, filter filter.LoaderFilter) (*bundle.Bundle, error) {
	b, err := bundle.NewCustomReader(bundle.NewFSLoaderWithRoot(fs, ".").WithFilter(filter)).
		WithCapabilities(io.Capabilities()).
		WithSkipBundleVerification(true).
		WithProcessAnnotations(true).
		WithLazyLoadingMode(false). // NB(sr): This is OPA's default, unless when using bundle plugins
		WithRegoVersion(ast.RegoV1).
		WithBundleName("regal").
		Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle from filesystem: %w", err)
	}

	return &b, nil
}

func mustLoad(fs fs.FS, filter filter.LoaderFilter) *bundle.Bundle {
	regalBundle, err := loadWithFilter(fs, filter)
	if err != nil {
		log.Fatal(err)
	}

	return regalBundle
}

func excludeTestFilter(path string, _ fs.FileInfo, _ int) bool {
	return isTestFile(path)
}

func excludeTestAndLanguageServerFilter(path string, _ fs.FileInfo, _ int) bool {
	return isTestFile(path) || strings.Contains(path, "lsp")
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.rego") && !strings.HasSuffix(path, "todo_test.rego")
}
