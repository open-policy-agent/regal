package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// opaLoadableExts are the file extensions the OPA loader accepts when
// given an explicit file path. Directory loads already ignore other types;
// once we expand a directory ourselves we must do the same.
var opaLoadableExts = map[string]struct{}{
	".rego": {},
	".yaml": {},
	".yml":  {},
	".json": {},
}

// FilterOPALoadPaths applies DefaultSkipDirectories when expanding workspace
// paths for loaders such as the OPA debugger. Without this, loading a
// workspace can fail after walking .git, .idea, or node_modules.
func FilterOPALoadPaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}

		if !info.IsDir() {
			out = append(out, p)
			continue
		}

		err = DefaultWalker(p).WithFilters(skipNonOPALoadable).Walk(func(path string) error {
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", p, err)
		}
	}

	return out, nil
}

func skipNonOPALoadable(path string, _ fs.DirEntry) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := opaLoadableExts[ext]

	return !ok
}
