package rules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/parse"
	rutil "github.com/open-policy-agent/regal/internal/util"
)

// Input represents the input for a linter evaluation.
type Input struct {
	// FileContent carries the string contents of each file
	FileContent map[string]string
	// Modules is the set of modules to lint.
	Modules map[string]*ast.Module
	// FileNames is used to maintain consistent order between runs.
	FileNames []string
}

type regoFile struct {
	name   string
	parsed *ast.Module
	raw    []byte
}

// NewInput creates a new Input from a set of modules.
func NewInput(fileContent map[string]string, modules map[string]*ast.Module) Input {
	return Input{
		FileContent: fileContent,
		FileNames:   util.KeysSorted(modules), // Maintain order across runs
		Modules:     modules,
	}
}

// InputFromPaths creates a new Input from a set of file or directory paths. Note that this function assumes that the
// paths point to valid Rego files. Use config.FilterIgnoredPaths to filter out unwanted content *before* calling this
// function. When the versionsMap is not nil/empty, files in a directory matching a key in the map will be parsed with
// the corresponding Rego version. If not provided, the file may be parsed multiple times in order to determine the
// version (best-effort and may include false positives).
func InputFromPaths(paths []string, prefix string, versionsMap map[string]ast.RegoVersion) (Input, error) {
	numPaths := len(paths)
	if numPaths == 1 && paths[0] == "-" {
		return inputFromStdin()
	}

	var versionedDirs []string
	if len(versionsMap) > 0 {
		// Sort directories by length, so that the most specific path is found first
		versionedDirs = util.KeysSorted(versionsMap)
		slices.Reverse(versionedDirs)
	}

	var wg sync.WaitGroup

	wg.Add(numPaths)

	errors := make([]error, numPaths)
	parsed := make([]*regoFile, numPaths)

	for i, path := range paths {
		go func(i int, path string) {
			opts := parse.ParserOptions()
			opts.RegoVersion = RegoVersionFromMap(versionsMap, strings.TrimPrefix(path, prefix), ast.RegoUndefined)

			if result, err := regoWithOpts(path, opts); err != nil {
				errors[i] = err
			} else {
				parsed[i] = result
			}

			wg.Done()
		}(i, path)
	}

	wg.Wait()

	if errors = rutil.Filter(errors, errNotNil); len(errors) > 0 {
		return Input{}, fmt.Errorf("failed to parse %d module(s) — first error: %w", len(errors), errors[0])
	}

	content := make(map[string]string, numPaths)
	modules := make(map[string]*ast.Module, numPaths)

	for _, file := range parsed {
		content[file.name] = util.ByteSliceToString(file.raw)
		modules[file.name] = file.parsed
	}

	return NewInput(content, modules), nil
}

// InputFromMap creates a new Input from a map of file paths to their contents.
// This function uses a vesrionsMap to determine the parser version for each
// file before parsing the module.
func InputFromMap(files map[string]string, versionsMap map[string]ast.RegoVersion) (Input, error) {
	content := make(map[string]string, len(files))
	modules := make(map[string]*ast.Module, len(files))
	prsopts := parse.ParserOptions()

	for path, fileContent := range files {
		prsopts.RegoVersion = RegoVersionFromMap(versionsMap, path, ast.RegoUndefined)

		mod, err := parse.ModuleWithOpts(path, fileContent, prsopts)
		if err != nil {
			return Input{}, fmt.Errorf("failed to parse module %s: %w", path, err)
		}

		content[path] = fileContent
		modules[path] = mod
	}

	return NewInput(content, modules), nil
}

// RegoVersionFromMap takes a mapping of file path prefixes, typically
// representing the roots of the project, and the expected Rego version for
// each. Using this, it finds the longest matching prefix for the given filename
// and returns the defaultVersion if to matching prefix is found.
func RegoVersionFromMap(
	versionsMap map[string]ast.RegoVersion,
	filename string,
	defaultVersion ast.RegoVersion,
) ast.RegoVersion {
	if len(versionsMap) == 0 {
		return defaultVersion
	}

	selectedVersion := defaultVersion
	dir := filepath.Dir(filename)

	var longestMatch int

	for versionedDir := range versionsMap {
		matchingVersionedDir := filepath.FromSlash(versionedDir)

		if strings.HasPrefix(dir, matchingVersionedDir) && len(versionedDir) >= longestMatch {
			// >= as the versioned dir might be "" for the project root
			longestMatch = len(versionedDir)
			selectedVersion = versionsMap[versionedDir]
		}
	}

	return selectedVersion
}

func regoWithOpts(path string, opts ast.ParserOptions) (*regoFile, error) {
	path = filepath.Clean(path)

	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	mod, err := parse.ModuleWithOpts(path, util.ByteSliceToString(bs), opts)
	if err != nil {
		return nil, err
	}

	return &regoFile{name: path, raw: bs, parsed: mod}, nil
}

func inputFromStdin() (Input, error) {
	// Ideally, we'd just pass the reader to OPA, but as the parser materializes
	// the input immediately anyway, there's currently no benefit to doing so.
	bs, err := io.ReadAll(os.Stdin)
	if err != nil {
		return Input{}, fmt.Errorf("failed to read from reader: %w", err)
	}

	policy := util.ByteSliceToString(bs)

	module, err := parse.ModuleUnknownVersionWithOpts("stdin", policy, parse.ParserOptions())
	if err != nil {
		return Input{}, fmt.Errorf("failed to parse module from stdin: %w", err)
	}

	return Input{
		FileContent: map[string]string{"stdin": policy},
		Modules:     map[string]*ast.Module{"stdin": module},
		FileNames:   []string{"stdin"},
	}, nil
}

// InputFromText creates a new Input from raw Rego text.
func InputFromText(fileName, text string) (Input, error) {
	return rutil.Wrap(InputFromTextWithOptions(fileName, text, parse.ParserOptions()))("can't create input from text")
}

// InputFromTextWithOptions creates a new Input from raw Rego text while respecting the provided options.
func InputFromTextWithOptions(fileName, text string, opts ast.ParserOptions) (Input, error) {
	mod, err := ast.ParseModuleWithOpts(fileName, text, opts)
	if err != nil {
		return Input{}, fmt.Errorf("failed to parse module: %w", err)
	}

	return NewInput(map[string]string{fileName: text}, map[string]*ast.Module{fileName: mod}), nil
}

func errNotNil(err error) bool {
	return err != nil
}
