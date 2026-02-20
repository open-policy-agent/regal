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
	name := info.Name()

	return info.IsDir() && (name == ".git" || name == ".idea" || name == "node_modules")
}

// Directories is a filter for filtering directories.
func Directories(_ string, info os.DirEntry) bool {
	return info.IsDir()
}

// Filenames filters files by their exact name, not including the path.
func Filenames(names ...string) Func {
	return func(_ string, info os.DirEntry) bool {
		return slices.ContainsFunc(names, func(name string) bool {
			return info.Name() == name
		})
	}
}

// Suffixes filters any path that has a suffix matching any of the provided suffixes.
func Suffixes(suffixes ...string) Func {
	return func(path string, _ os.DirEntry) bool {
		return slices.ContainsFunc(suffixes, func(suffix string) bool {
			return strings.HasSuffix(path, suffix)
		})
	}
}

// Not negates the result of the provided filter, so that e.g.
// Not(Directories) will return true for files and false for directories.
func Not(filter Func) Func {
	return func(path string, info os.DirEntry) bool {
		return !filter(path, info)
	}
}

func RegoTests(path string, info os.DirEntry) bool {
	return strings.HasSuffix(path, "_test.rego") && info.Name() != "todo_test.rego"
}

func NotRego(path string, _ os.DirEntry) bool {
	return !strings.HasSuffix(path, ".rego") && !strings.HasSuffix(path, ".rq")
}
