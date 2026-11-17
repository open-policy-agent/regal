package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/loader"
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
//
// When the OPA loader walks a directory itself, data files are loaded under
// the path of the directory they are in (a/data.json becomes data.a). Passing
// the expanded file paths would load them all at the root of data instead, so
// each data file gets a "prefix:path" entry that keeps the same layout. A
// directory name containing a '.' or ':' cannot be written as a prefix, so a
// root with data in such a directory is passed through unexpanded.
func FilterOPALoadPaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))

	for _, p := range paths {
		prefix, path := loader.SplitPrefix(p)

		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if !info.IsDir() {
			out = append(out, p)

			continue
		}

		expanded, ok, err := expandDirectory(prefix, path)
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", path, err)
		}

		if !ok {
			out = append(out, p)

			continue
		}

		out = append(out, expanded...)
	}

	return out, nil
}

func expandDirectory(prefix []string, root string) ([]string, bool, error) {
	var out []string

	representable := true

	err := DefaultWalker(root).WithFilters(skipNonOPALoadable).Walk(func(path string) error {
		if strings.EqualFold(filepath.Ext(path), ".rego") {
			// Modules are placed by their package, not by their location.
			out = append(out, path)

			return nil
		}

		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}

		parts := prefix
		if rel != "." {
			parts = append(slices.Clip(parts), strings.Split(rel, string(filepath.Separator))...)
		}

		if len(parts) == 0 {
			out = append(out, path)

			return nil
		}

		for _, part := range parts {
			if part == "" || strings.ContainsAny(part, ".:") {
				representable = false

				return nil
			}
		}

		out = append(out, strings.Join(parts, ".")+":"+path)

		return nil
	})

	return out, representable, err
}

func skipNonOPALoadable(path string, _ fs.DirEntry) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := opaLoadableExts[ext]

	return !ok
}
