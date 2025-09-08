package fixes

import (
	"cmp"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/config"
)

type DirectoryPackageMismatch struct{}

func (*DirectoryPackageMismatch) Name() string {
	return "directory-package-mismatch"
}

// For now, just handle the "normal" set of characters, plus hyphens.
// We can broaden this later, but we should avoid characters that may have
// special meaning in files and directories.
var regularName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// Fix moves a file to the correct directory based on the package name, and relative to the base
// directory provided in opts. If the file is already in the correct directory, no action is taken.
func (d *DirectoryPackageMismatch) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
	pkgPath, err := getPackagePathDirectory(fc, opts.Config)
	if err != nil {
		return nil, err
	}

	rootPath := filepath.Clean(cmp.Or(opts.BaseDir, filepath.Dir(fc.Filename)))
	newPath := filepath.Join(rootPath, pkgPath, filepath.Base(fc.Filename))

	if newPath == fc.Filename {
		return nil, nil // File is where it should be. We are done!
	}

	return []FixResult{{
		Title:    d.Name(),
		Root:     rootPath,
		Contents: fc.Contents,
		Rename: &Rename{
			FromPath: fc.Filename,
			ToPath:   newPath, // TODO: should we check that this is relative to base somewhere?
		},
	}}, nil
}

func getPackagePathDirectory(fc *FixCandidate, config *config.Config) (string, error) {
	module, err := ast.ParseModule(fc.Filename, fc.Contents)
	if err != nil {
		return "", err
	}

	parts := make([]string, len(module.Package.Path)-1)
	excludeTestSuffix := shouldExcludeTestSuffix(config)
	pathWithoutData := module.Package.Path[1:]

	for i, part := range pathWithoutData {
		text := strings.Trim(part.Value.String(), "\"")

		if !regularName.MatchString(text) {
			return "", fmt.Errorf("can only handle [a-zA-Z0-9_-] characters in package name, got: %s", text)
		}

		if i == len(pathWithoutData)-1 && excludeTestSuffix {
			text = strings.TrimSuffix(text, "_test")
		}

		parts[i] = text
	}

	return filepath.Join(parts...), nil
}

func shouldExcludeTestSuffix(config *config.Config) bool {
	if config != nil {
		if category, ok := config.Rules["idiomatic"]; ok {
			if rule, ok := category["directory-package-mismatch"]; ok {
				if exclude, ok := rule.Extra["exclude-test-suffix"].(bool); ok {
					return exclude
				}
			}
		}
	}

	// this is the default, and this should be unreachable provided that the
	// provided configuration was included (which it always would be)
	return true
}
