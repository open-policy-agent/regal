package files

import (
	"fmt"
	"io/fs"
	"maps"
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
// directory name containing a '.' or ':' cannot be written as a prefix, so the
// top-level directory holding it is passed through unexpanded (see expandDirectory).
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

// expandDirectory walks root and returns its loadable files, with data files as
// "prefix:path" entries. A name with a '.' or ':' cannot be a prefix, so the
// top-level directory holding it is returned whole for the loader to walk. If
// that name is itself top-level, the bool is false and nothing is expanded.
func expandDirectory(prefix []string, root string) ([]string, bool, error) {
	var out, tops []string

	whole, passthrough := true, map[string]bool{}

	add := func(top, entry string) {
		out, tops = append(out, entry), append(tops, top)
	}

	err := NewWalker(root).WithFilter(isOPALoadable).Walk(func(path string) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		dirs := strings.Split(rel, string(filepath.Separator))
		if slices.Contains(dirs[:len(dirs)-1], ".github") {
			// Workflow files are not data, and the name cannot be a prefix.
			return nil
		}

		top, _, nested := strings.Cut(rel, string(filepath.Separator))
		if !nested {
			top = ""
		}

		if strings.EqualFold(filepath.Ext(path), ".rego") {
			// Modules are placed by their package, not by their location.
			add(top, path)

			return nil
		}

		parts := prefix
		if dir := filepath.Dir(rel); dir != "." {
			parts = append(slices.Clip(parts), strings.Split(dir, string(filepath.Separator))...)
		}

		if len(parts) == 0 {
			add(top, path)

			return nil
		}

		for _, part := range parts {
			if part == "" || strings.ContainsAny(part, ".:") {
				if top == "" || strings.ContainsAny(top, ".:") {
					whole = false
				}

				passthrough[top] = true

				return nil
			}
		}

		add(top, strings.Join(parts, ".")+":"+path)

		return nil
	})
	if err != nil || !whole {
		return nil, false, err
	}

	res := make([]string, 0, len(out))

	for i, entry := range out {
		if !passthrough[tops[i]] {
			res = append(res, entry)
		}
	}

	for _, top := range slices.Sorted(maps.Keys(passthrough)) {
		res = append(res, strings.Join(append(slices.Clip(prefix), top), ".")+":"+filepath.Join(root, top))
	}

	return res, true, nil
}

func isOPALoadable(path string, d fs.DirEntry) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := opaLoadableExts[ext]

	return ok && !d.IsDir()
}
