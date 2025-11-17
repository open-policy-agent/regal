package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
)

func FilterIgnoredPaths(paths, ignore []string, checkExists bool, pathPrefix string) (filtered []string, err error) {
	// - special case for stdin, return as is
	if len(paths) == 0 || len(paths) == 1 && paths[0] == "-" || !checkExists && len(ignore) == 0 {
		return paths, nil
	}

	if checkExists {
		for _, path := range paths {
			filtered, err = files.DefaultWalkReducer(path, filtered).
				WithFilters(filter.NotRego).
				WithStatBeforeWalk(true).
				Reduce(files.PathAppendReducer)
			if err != nil {
				return nil, fmt.Errorf("failed to filter paths:\n%w", err)
			}
		}

		paths = filtered
	}

	// Use forward slash since all paths are normalized to forward slashes for glob matching
	return filterPaths(paths, ignore, util.EnsureSuffix(pathPrefix, "/"))
}

func filterPaths(policyPaths, ignore []string, pathPrefix string) ([]string, error) {
	patterns, err := compilePatterns(ignore)
	if err != nil {
		return nil, fmt.Errorf("failed to compile ignore patterns: %w", err)
	}

	filtered := make([]string, 0, len(policyPaths))

outer:
	for _, f := range policyPaths {
		for _, pattern := range patterns {
			if excludeFile(pattern, f, pathPrefix) {
				continue outer
			}
		}

		filtered = append(filtered, f)
	}

	return filtered, nil
}

// excludeFile imitates the pattern matching of .gitignore files
// See `exclusion.rego` for details on the implementation.
func excludeFile(pattern glob.Glob, filename, pathPrefix string) bool {
	// Normalize path separators to forward slashes for consistent glob matching
	filename = filepath.ToSlash(filename)
	if pathPrefix != "" {
		filename = strings.TrimPrefix(strings.TrimPrefix(filename, filepath.ToSlash(pathPrefix)), "/")
	}

	return pattern.Match(filename)
}

func compilePatterns(patterns []string) ([]glob.Glob, error) {
	compiled := make([]glob.Glob, 0, len(patterns))

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		n := len(pattern)
		// Internal slashes means path is relative to root, otherwise it can
		// appear anywhere in the directory (--> **/)
		if !strings.Contains(pattern[:n-1], "/") {
			pattern = "**/" + pattern
		}

		pattern = strings.TrimPrefix(pattern, "/")

		ps := []string{pattern}
		if noPrefix, ok := strings.CutPrefix(pattern, "**/"); ok {
			ps = append(ps, noPrefix)
		}

		var ps1 []string

		for _, p := range ps {
			switch {
			case strings.HasSuffix(p, "/"):
				ps1 = append(ps1, p+"**")
			case !strings.HasSuffix(p, "**") && !strings.HasSuffix(p, ".rego"):
				ps1 = append(ps1, p, p+"/**")
			default:
				ps1 = append(ps1, p)
			}
		}

		// Loop through patterns and return true on first match
		for _, p := range ps1 {
			g, err := glob.Compile(p, '/')
			if err != nil {
				return nil, fmt.Errorf("failed to compile pattern %s: %w", p, err)
			}

			compiled = append(compiled, g)
		}
	}

	return compiled, nil
}
