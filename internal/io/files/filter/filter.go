package filter

import (
	"io/fs"
	"os"
	"slices"
	"strings"
)

// Func is a function that filters files and directories during a walk.
type Func func(path string, info os.DirEntry) bool

// DefaultSkipDirectories is a default skip function that skips common directories
// that are not relevant for most file walks, such as .git, .idea, and node_modules.
func DefaultSkipDirectories(_ string, info fs.DirEntry) bool {
	return dirWithName(info, ".git", ".idea", "node_modules")
}

// File is a filter for filtering files (excluding directories).
func File(_ string, info os.DirEntry) bool {
	return !info.IsDir()
}

// RegoNoTests filters Rego files excluding test files, except for the special case "todo_test.rego".
func RegoNoTests(path string, d os.DirEntry) bool {
	return !d.IsDir() && strings.HasSuffix(path, ".rego") &&
		(!strings.HasSuffix(path, "_test.rego") || d.Name() == "todo_test.rego")
}

// Rego filters all Rego files, including test files.
func Rego(path string, d os.DirEntry) bool {
	return !d.IsDir() && (strings.HasSuffix(path, ".rego") || strings.HasSuffix(path, ".rq"))
}

// Config filters config.yaml and .regal.yaml files.
func Config(_ string, d os.DirEntry) bool {
	return AnyFileName(d, "config.yaml", ".regal.yaml")
}

// Input filters input.json and input.yaml files.
func Input(_ string, d os.DirEntry) bool {
	return AnyFileName(d, "input.json", "input.yaml")
}

// Manifest filters .manifest files.
func Manifest(_ string, d os.DirEntry) bool {
	return !d.IsDir() && d.Name() == ".manifest"
}

// JSON filters .json files.
func JSON(path string, d os.DirEntry) bool {
	return !d.IsDir() && strings.HasSuffix(path, ".json")
}

// AnyFileName is a helper function to check if d is a file named as any of the provided names.
func AnyFileName(d os.DirEntry, names ...string) bool {
	return !d.IsDir() && slices.Contains(names, d.Name())
}

func dirWithName(d os.DirEntry, names ...string) bool {
	return d.IsDir() && slices.Contains(names, d.Name())
}
