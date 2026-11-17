package io

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/io/paths"
	"github.com/open-policy-agent/regal/internal/util"

	_ "github.com/open-policy-agent/regal/pkg/builtins/regal"
)

type (
	readWriteCloser struct {
		in  io.ReadCloser
		out io.WriteCloser
	}
	namedModules = map[string]*ast.Module
)

var (
	OPACapabilities = sync.OnceValue(opaCapabilities)
	Capabilities    = sync.OnceValue(capabilities)
)

func NewReadWriteCloser(in io.ReadCloser, out io.WriteCloser) readWriteCloser {
	return readWriteCloser{in: in, out: out}
}

func (rwc readWriteCloser) Read(p []byte) (int, error) {
	return util.Wrap(rwc.in.Read(p))("failed to read from stdin")
}

func (rwc readWriteCloser) Write(p []byte) (int, error) {
	return util.Wrap(rwc.out.Write(p))("failed to write to stdout")
}

func (rwc readWriteCloser) Close() error {
	return errors.Join(rwc.in.Close(), rwc.out.Close())
}

// Getwd returns the current working directory, or an empty string if it cannot be determined.
func Getwd() string {
	wd, _ := os.Getwd()

	return wd
}

// CloseIgnore closes handle ignoring errors, mainly for deferred cleanup.
func CloseIgnore(handle io.Closer) {
	if handle != nil {
		_ = handle.Close()
	}
}

func Dir(path string) string {
	if IsDir(path) {
		return path
	}

	return filepath.Dir(path)
}

func IsDir(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

func IsFile(path string) bool {
	info, err := os.Stat(path)

	return err == nil && !info.IsDir()
}

func Exists(path string) bool {
	return util.OnlyError(os.Stat(path)) == nil
}

// WithOpen opens the file at path, defers close, and invokes f with the *os.File.
// It returns the result and preserves the original error semantics from the previous location.
func WithOpen[T any](path string, f func(*os.File) (T, error)) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T

		return zero, fmt.Errorf("failed to open file '%s': %w", path, err)
	}
	defer file.Close()

	return f(file)
}

// WithCreateRecursive creates a file at the given path, ensuring that all parent directories
// are created recursively. It then calls the provided function with the created file as an argument
// before closing the file.
func WithCreateRecursive(path string, fn func(f *os.File) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o770); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer CloseIgnore(file)

	return fn(file)
}

// FindManifestLocations walks the file system rooted at root, and returns the
// *relative* paths of directories containing a .manifest file.
func FindManifestLocations(root string) (found []string, _ error) {
	return files.NewWalker(root).
		WithFilter(filter.Manifest).
		Reduce(found, func(path string, acc []string) ([]string, error) {
			rel, err := filepath.Rel(root, path)
			if err == nil {
				acc = append(acc, filepath.Dir(rel))
			}

			return acc, err
		})
}

func ModulesFromCustomRuleFS(customRuleFS fs.FS, root string) (namedModules, error) {
	return files.NewWalker(root).
		WithFileSystem(customRuleFS).
		WithFilter(filter.RegoNoTests).
		Reduce(make(namedModules), func(path string, mods namedModules) (_ namedModules, err error) {
			var bs []byte

			if bs, err = fs.ReadFile(customRuleFS, path); err == nil {
				mods[path], err = ast.ParseModule(path, outil.ByteSliceToString(bs))
			}

			return mods, err
		})
}

// DirCleanUpPaths will, for a given target file, list all the dirs that would
// be empty if the target file was deleted.
func DirCleanUpPaths(target string, preserve []string) ([]string, error) {
	preserveDirs := util.NewSet(preserve...)

	for _, p := range preserve {
		for {
			p = filepath.Dir(p)
			if p == "." || p == "/" || p == paths.Separator ||
				p == filepath.VolumeName(p) || preserveDirs.Contains(p) {
				break
			}

			preserveDirs.Add(p)
		}
	}

	var dirs []string

	dir := filepath.Dir(target)

	for !preserveDirs.Contains(dir) {
		if !strings.Contains(dir, paths.Separator) {
			break
		}

		if !IsDir(dir) {
			return nil, fmt.Errorf("expected directory, got file %s", dir)
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		empty := true

		for _, file := range files {
			// exclude the target
			abs := filepath.Join(dir, file.Name())
			if abs == target {
				continue
			}

			// exclude any other marked dirs
			if file.IsDir() && len(dirs) > 0 {
				if dirs[len(dirs)-1] == abs {
					continue
				}
			}

			empty = false

			break
		}

		if !empty {
			break
		}

		dirs = append(dirs, dir)
		dir = filepath.Dir(dir)
	}

	return dirs, nil
}

// An interestng side-effect of pushing our builtins into ast.RegisterBuiltin is that
// they end up in "OPA's" list of builtins. Hence why we reverse that here.
func capabilities() *ast.Capabilities {
	cpy := *ast.CapabilitiesForThisVersion()

	return &cpy
}

func opaCapabilities() *ast.Capabilities {
	cpy := *ast.CapabilitiesForThisVersion()
	cpy.Builtins = util.Filter(cpy.Builtins, func(b *ast.Builtin) bool {
		return !strings.HasPrefix(b.Name, "regal.")
	})

	return &cpy
}
