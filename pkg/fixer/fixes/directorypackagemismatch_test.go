package fixes

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestFixDirectoryPackageMismatch(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		name              string
		baseDir           string
		contents          string
		expected          *FixResult
		wantErr           string
		includeTestSuffix bool // inverse of exclude-text-suffix to avoid providing the default everywhere
	}{
		"files are moved according to their package": {
			name:     "/root/main.rego",
			baseDir:  "/root",
			contents: "package main",
			expected: &FixResult{
				Contents: "package main",
				Rename: &Rename{
					FromPath: "/root/main.rego",
					ToPath:   "/root/main/main.rego",
				},
			},
		},
		"files are moved from nested dirs": {
			name:     "/root/foo/bar.rego",
			baseDir:  "/root",
			contents: "package bar",
			expected: &FixResult{
				Contents: "package bar",
				Rename: &Rename{
					FromPath: "/root/foo/bar.rego",
					ToPath:   "/root/bar/bar.rego",
				},
			},
		},
		"files are moved with nested pkgs": {
			name:     "/root/foo/bar.rego",
			baseDir:  "/root",
			contents: "package bar.bar",
			expected: &FixResult{
				Contents: "package bar.bar",
				Rename: &Rename{
					FromPath: "/root/foo/bar.rego",
					ToPath:   "/root/bar/bar/bar.rego",
				},
			},
		},
		"package names with hyphens create directories with hyphens": {
			name:     "/root/foo.rego",
			baseDir:  "/root",
			contents: `package foo["bar-baz"].qux`,
			expected: &FixResult{
				Contents: `package foo["bar-baz"].qux`,
				Rename: &Rename{
					FromPath: "/root/foo.rego",
					ToPath:   "/root/foo/bar-baz/qux/foo.rego",
				},
			},
		},
		"package with special characters returns an error": {
			name:     "/root/foo.rego",
			baseDir:  "/root",
			contents: `package foo["bar/baz"].qux`,
			wantErr:  "can only handle [a-zA-Z0-9_-] characters in package name, got: bar/baz",
		},
		"package names with _test suffix are by default treated as if without the suffix": {
			name:     "/root/foo_test.rego",
			baseDir:  "/root",
			contents: `package foo_test`,
			expected: &FixResult{
				Contents: `package foo_test`,
				Rename: &Rename{
					FromPath: "/root/foo_test.rego",
					ToPath:   "/root/foo/foo_test.rego",
				},
			},
		},
		"package names with _test suffix are handled when configured": {
			name:     "/root/foo_test.rego",
			baseDir:  "/root",
			contents: `package foo_test`,
			expected: &FixResult{
				Contents: `package foo_test`,
				Rename: &Rename{
					FromPath: "/root/foo_test.rego",
					ToPath:   "/root/foo_test/foo_test.rego",
				},
			},
			includeTestSuffix: true,
		},
	}

	for testCase, tc := range cases {
		t.Run(testCase, func(t *testing.T) {
			t.Parallel()

			dpm := DirectoryPackageMismatch{}
			cnd := &FixCandidate{Filename: filepath.FromSlash(tc.name), Contents: tc.contents}
			opt := &RuntimeOptions{
				BaseDir: filepath.FromSlash(tc.baseDir),
				Config:  configWithExcludeTestSuffix(!tc.includeTestSuffix),
			}

			fr, err := dpm.Fix(cnd, opt)
			if tc.wantErr != "" {
				testutil.ErrMustContain(err, tc.wantErr)(t)

				return
			}

			if err != nil {
				t.Errorf("failed to fix: %v", err)
			}

			if len(fr) > 2 {
				t.Fatal("0 or 1 fix result expected")
			}

			if len(fr) == 0 && tc.expected != nil {
				t.Fatalf("expected fix result, got none")
			}

			fixResult := fr[0]

			if fixResult.Contents != tc.expected.Contents {
				t.Fatalf("expected %s, got %s", tc.expected.Contents, fr[0].Contents)
			}

			if tc.expected.Rename != nil {
				if fixResult.Rename == nil {
					t.Fatal("expected rename to be non-nil, got nil")
				}

				if fixResult.Rename.FromPath != tc.expected.Rename.FromPath {
					t.Fatalf("expected from path to be %s, got %s", tc.expected.Rename.FromPath, fixResult.Rename.FromPath)
				}

				if exp, got := fixResult.Rename.ToPath, filepath.FromSlash(tc.expected.Rename.ToPath); exp != got {
					t.Fatalf("expected to path to be %s, got %s", exp, got)
				}
			}
		})
	}
}

func configWithExcludeTestSuffix(exclude bool) *config.Config {
	return &config.Config{Rules: map[string]config.Category{"idiomatic": {
		"directory-package-mismatch": config.Rule{
			Level: "ignore",
			Extra: map[string]any{"exclude-test-suffix": exclude},
		},
	}}}
}
