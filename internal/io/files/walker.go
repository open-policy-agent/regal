package files

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/open-policy-agent/regal/internal/io/files/filter"
)

type (
	// Walker is a utility for walking directories and files, applying
	// provided filters before processing each file or directory.
	Walker struct {
		// root is the root directory to start walking from.
		root string
		// filesystem is the file system to walk. If nil, the local file system is used.
		filesystem fs.FS
		// filter sets the filter function to apply to each file or directory.
		// If the filter returns false, the file or directory is skipped.
		filter filter.Func
		// skipFn is a special filter that allows skipping entire directory
		// entire directory trees (like .git, node_modules, etc.).
		skipFn filter.Func
		// statRoot indicates whether to stat the root directory before walking.
		statRoot bool
	}
)

var errEmptyRoot = errors.New("root path is empty")

// NewWalker returns a new [Walker] with default / recommended settings,
// which means directories that are known to be irrelevant for Regal (e.g.
// .git, node_modules, etc.) aren't traversed at all.
func NewWalker(root string) Walker {
	return Walker{root: root, skipFn: filter.DefaultSkipDirectories}
}

// WithFilter adds a filter to the [Walker].
func (fw Walker) WithFilter(filter filter.Func) Walker {
	fw.filter = filter

	return fw
}

// WithStatBeforeWalk sets whether to stat the root directory before walking.
// The underlying [filepath.WalkDir] implementation panics on non-existent paths,
// so this is useful when the input isn't guaranteed to exist.
func (fw Walker) WithStatBeforeWalk(statRoot bool) Walker {
	fw.statRoot = statRoot

	return fw
}

// WithFileSystem sets the file system to use for the [Walker]. If not set, the local file system is used.
func (fw Walker) WithFileSystem(fs fs.FS) Walker {
	fw.filesystem = fs
	if path.IsAbs(fw.root) {
		fw.root = "."
	}

	return fw
}

// Walk walks the file system rooted at the [Walker]'s root, calling f for each file
// not filtered out by the [Walker]'s filters.
func (fw Walker) Walk(f func(string) error) error {
	if err := fw.validate(); err != nil {
		return err
	}

	return fw.walkDir(fw.root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
		case fw.skipFn != nil && fw.skipFn(path, d):
			err = fs.SkipDir
		case fw.filter != nil && fw.filter(path, d):
			err = f(path)
		}

		return err
	})
}

// Reduce walks the file system rooted at the [Walker]'s root, calling f for each file.
// The function f receives the path of the file and the current value of the
// accumulator. It returns the new value of the accumulator and an error if any.
// The [Walker]'s filters are applied before calling f.
func (fw Walker) Reduce[T any](acc T, f func(string, T) (T, error)) (T, error) {
	if err := fw.validate(); err != nil {
		return acc, err
	}

	return acc, fw.walkDir(fw.root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
		case fw.skipFn != nil && fw.skipFn(path, d):
			err = fs.SkipDir
		case fw.filter != nil && fw.filter(path, d):
			acc, err = f(path, acc)
		}

		return err
	})
}

func (fw Walker) walkDir(path string, f fs.WalkDirFunc) error {
	if fw.filesystem == nil {
		return filepath.WalkDir(fw.root, f)
	}

	return fs.WalkDir(fw.filesystem, path, f)
}

func (fw Walker) validate() (err error) {
	switch {
	case fw.root == "":
		return errEmptyRoot
	case fw.statRoot:
		_, err = os.Stat(fw.root)
	}

	return err
}
