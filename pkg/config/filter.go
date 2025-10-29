package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"

	"github.com/open-policy-agent/opa/v1/bundle"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
)

func FilterIgnoredPaths(paths, ignore []string, checkExists bool, pathPrefix string) (filtered []string, err error) {
	// - special case for stdin, return as is
	if len(paths) == 1 && paths[0] == "-" || !checkExists && len(ignore) == 0 {
		return paths, nil
	}

	if checkExists {
		for _, path := range paths {
			filtered, err = files.DefaultWalkReducer(path, filtered).
				WithFilters(filter.Not(filter.Suffixes(bundle.RegoExt))).
				WithStatBeforeWalk(true).
				Reduce(files.PathAppendReducer)
			if err != nil {
				return nil, fmt.Errorf("failed to filter paths:\n%w", err)
			}
		}

		return filterPaths(filtered, ignore, util.EnsureSuffix(pathPrefix, string(os.PathSeparator)))
	}

	return filterPaths(paths, ignore, util.EnsureSuffix(pathPrefix, string(os.PathSeparator)))
}

func filterPaths(policyPaths []string, ignore []string, pathPrefix string) ([]string, error) {
	filtered := make([]string, 0, len(policyPaths))

outer:
	for _, f := range policyPaths {
		for _, pattern := range ignore {
			if pattern == "" {
				continue
			}

			excluded, err := excludeFile(pattern, f, pathPrefix)
			if err != nil {
				return nil, fmt.Errorf("failed to check for exclusion using pattern %s: %w", pattern, err)
			}

			if excluded {
				continue outer
			}
		}

		filtered = append(filtered, f)
	}

	return filtered, nil
}

// excludeFile imitates the pattern matching of .gitignore files
// See `exclusion.rego` for details on the implementation.
func excludeFile(pattern, filename, pathPrefix string) (bool, error) {
	n := len(pattern)

	if pathPrefix != "" {
		filename = strings.TrimPrefix(filename, pathPrefix)
	}

	// Normalize path separators to forward slashes for consistent glob matching
	filename = filepath.ToSlash(filename)

	// Internal slashes means path is relative to root, otherwise it can
	// appear anywhere in the directory (--> **/)
	if !strings.Contains(pattern[:n-1], "/") {
		pattern = "**/" + pattern
	}

	// Leading slash?
	pattern = strings.TrimPrefix(pattern, "/")

	// Leading double-star?
	ps := []string{pattern}
	if noPrefix, ok := strings.CutPrefix(pattern, "**/"); ok {
		ps = append(ps, noPrefix)
	}

	var ps1 []string

	// trailing slash?
	for _, p := range ps {
		switch {
		case strings.HasSuffix(p, "/"):
			ps1 = append(ps1, p+"**")
		case !strings.HasSuffix(p, "**"):
			ps1 = append(ps1, p, p+"/**")
		default:
			ps1 = append(ps1, p)
		}
	}

	// Loop through patterns and return true on first match
	for _, p := range ps1 {
		g, err := glob.Compile(p, '/')
		if err != nil {
			return false, fmt.Errorf("failed to compile pattern %s: %w", p, err)
		}

		if g.Match(filename) {
			return true, nil
		}
	}

	return false, nil
}
