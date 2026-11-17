package files_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/test/must"
)

// Benchmark to demonstrate that our [files.Walker] convenience
// incurs negligible overhead compared to the [filepath.WalkDir] and
// [fs.WalkDir] implementations.
//
// Walker.Walk            46549 ns/op    6728 B/op      43 allocs/op
// filepath.WalkDir       46874 ns/op    6728 B/op      43 allocs/op
// Walker.Walk_with_FS    50022 ns/op    6896 B/op      50 allocs/op
// fs.WalkDir             50373 ns/op    6896 B/op      50 allocs/op
func BenchmarkWalkerWalkVsFilepathWalkDir(b *testing.B) {
	dir := b.TempDir()
	sub := filepath.Join(dir, "sub1", "sub2", "sub3")
	dfs := os.DirFS(dir)

	must.Equal(b, nil, os.MkdirAll(sub, 0o755))
	must.Equal(b, nil, os.WriteFile(filepath.Join(sub, ".regal.yaml"), []byte("content"), 0o644))

	b.Run("Walker.Walk", func(b *testing.B) {
		for b.Loop() {
			must.Equal(b, nil, files.NewWalker(dir).WithFilter(filter.Config).Walk(walkerWalkDir))
		}
	})

	b.Run("filepath.WalkDir", func(b *testing.B) {
		for b.Loop() {
			must.Equal(b, nil, filepath.WalkDir(dir, stdlibWalkDir))
		}
	})

	b.Run("Walker.Walk with FS", func(b *testing.B) {
		for b.Loop() {
			must.Equal(b, nil, files.NewWalker(dir).WithFileSystem(dfs).WithFilter(filter.Config).Walk(walkerWalkDir))
		}
	})

	b.Run("fs.WalkDir", func(b *testing.B) {
		for b.Loop() {
			must.Equal(b, nil, fs.WalkDir(dfs, ".", stdlibWalkDir))
		}
	})
}

func walkerWalkDir(string) error { return nil }

func stdlibWalkDir(path string, d os.DirEntry, err error) error {
	// apply same filter to make the comparison fair
	if err == nil {
		_ = filter.Config(path, d)
	}

	return err
}
