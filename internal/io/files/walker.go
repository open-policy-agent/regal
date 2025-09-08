package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/open-policy-agent/regal/internal/io/files/filter"
)

// Walker is a utility for walking directories and files, applying
// provided filters before processing each file or directory.
type Walker struct {
	// root is the root directory to start walking from.
	root string
	// filters are the filters to apply to each file or directory.
	// If any filter returns true, the file or directory is skipped.
	filters []filter.Func
	// skipFunc is a special filter that allows skipping entire directory
	// entire directory trees (like .git, node_modules, etc.).
	skipFunc filter.Func
	// statRoot indicates whether to stat the root directory before walking.
	statRoot bool
}

// Note(anderseknert):
// It would have been nice if Walker could both Walk and Reduce instead of
// having this separation. The reason we don't is that the reducer needs a
// generic type argument for the accumulator, and this needs to be provided
// when creating the instance. Having clients forced to provide an accumulator
// type just for walking is not a great experience, so for now we split the two.
// Perhaps we'll find a better way to do it in the future.

// WalkReducer extends FileWalker to allow reducing the results of the walk.
type WalkReducer[T any] struct {
	*Walker

	initial T
}

// NewWalker creates a new FileWalker with the specified root directory.
func NewWalker(root string) *Walker {
	return &Walker{
		root: root,
	}
}

// DefaultWalker returns a new Walker with default / recommended settings,
// which means only files are walked and that directories that are known to be
// irrelevant for Regal (e.g. .git, node_modules, etc.) aren't traversed at all.
func DefaultWalker(root string) *Walker {
	return NewWalker(root).WithSkipFunc(filter.DefaultSkipDirectories).WithFilters(filter.Directories)
}

// WithFilters adds filters to the FileWalker.
func (fw *Walker) WithFilters(filters ...filter.Func) *Walker {
	fw.filters = append(fw.filters, filters...)

	return fw
}

// WithSkipFunc sets the skip function for the FileWalker. This is a special
// filter that allows skipping traversal of entire directory trees (like
// .git, node_modules, etc.).
func (fw *Walker) WithSkipFunc(skipFunc filter.Func) *Walker {
	fw.skipFunc = skipFunc

	return fw
}

// WithStatBeforeWalk sets whether to stat the root directory before walking.
// The underlying filepath.WalkDir implementation panics on non-existent paths,
// so this is useful when the input isn't guaranteed to exist.
func (fw *Walker) WithStatBeforeWalk(statRoot bool) *Walker {
	fw.statRoot = statRoot

	return fw
}

// Walk walks the file system rooted at the root, calling f for each file
// not filtered out by the filters.
func (fw *Walker) Walk(f func(string) error) error {
	if err := fw.validate(); err != nil {
		return err
	}

	return filepath.WalkDir(fw.root, fw.walker(f))
}

// WalkFS walks the file system rooted at the root using the provided fs.FS,
// calling f for each file not filtered out by the filters.
func (fw *Walker) WalkFS(target fs.FS, f func(string) error) error {
	if err := fw.validate(); err != nil {
		return err
	}

	return fs.WalkDir(target, fw.root, fw.walker(f))
}

func (fw *Walker) validate() error {
	if fw.root == "" {
		return errors.New("root path is empty")
	}

	if fw.statRoot {
		if _, err := os.Stat(fw.root); err != nil {
			return err
		}
	}

	return nil
}

func (fw *Walker) walker(f func(string) error) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", path, err)
		}

		if fw.skipFunc != nil && fw.skipFunc(path, d) {
			return fs.SkipDir
		}

		for _, filter := range fw.filters {
			if filter(path, d) {
				return nil
			}
		}

		return f(path)
	}
}

// NewWalkReducer creates a new FileWalkerReducer with the specified root
// directory and initial value for the accumulator.
func NewWalkReducer[T any](root string, into T) *WalkReducer[T] {
	return &WalkReducer[T]{
		Walker:  &Walker{root: root},
		initial: into,
	}
}

// DefaultWalkReducer returns a new WalkReducer embedding a DefaultWalker
// sharing its default recommended settings.
func DefaultWalkReducer[T any](root string, into T) *WalkReducer[T] {
	return &WalkReducer[T]{
		Walker:  DefaultWalker(root),
		initial: into,
	}
}

// WithFilters adds filters to the FileWalkerReducer.
func (fwr *WalkReducer[T]) WithFilters(filters ...filter.Func) *WalkReducer[T] {
	fwr.filters = append(fwr.filters, filters...)

	return fwr
}

// WithSkipFunc sets the skip function for the FileWalkerReducer.
func (fwr *WalkReducer[T]) WithSkipFunc(skipFunc filter.Func) *WalkReducer[T] {
	fwr.skipFunc = skipFunc

	return fwr
}

// WithStatBeforeWalk sets whether to stat the root directory before walking.
// The underlying filepath.WalkDir implementation panics on non-existent paths,
// so this is useful when the input isn't guaranteed to exist.
func (fwr *WalkReducer[T]) WithStatBeforeWalk(statRoot bool) *WalkReducer[T] {
	fwr.statRoot = statRoot

	return fwr
}

// Reduce walks the file system rooted at fwr.Root, calling f for each file.
// The function f receives the path of the file and the current value of the
// accumulator. It returns the new value of the accumulator and an error if any.
func (fwr *WalkReducer[T]) Reduce(f func(string, T) (T, error)) (T, error) {
	curr := fwr.initial
	err := fwr.Walk(func(path string) (err error) {
		curr, err = f(path, curr)

		return err
	})

	return curr, err
}

// ReduceFS walks the file system rooted at fwr.Root using the provided fs.FS,
// calling f for each file. The function f receives the path of the file and
// the current value of the accumulator. It returns the new value of the
// accumulator and an error if any.
// NOTE: Unlike that path passed to f is always /-separated when walking an FS
// implementation (as opposed to a directory, where the separator is OS-specific).
func (fwr *WalkReducer[T]) ReduceFS(target fs.FS, f func(string, T) (T, error)) (T, error) {
	curr := fwr.initial
	err := fwr.WalkFS(target, func(path string) (err error) {
		curr, err = f(path, curr)

		return err
	})

	return curr, err
}

// PathAppendReducer is a simple reducer function that appends the
// given path to the current slice of strings.
func PathAppendReducer(path string, curr []string) ([]string, error) {
	return append(curr, path), nil
}
