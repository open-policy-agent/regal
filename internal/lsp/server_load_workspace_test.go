package lsp

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestLoadWorkspaceContents(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		files       map[string]string
		cachedFiles []string
		// override contents of the server cache
		cachedContent      map[string]string
		unreadableFiles    []string
		newOnly            bool
		expectChangedFiles []string
		expectFailedFiles  []string
	}{
		"loads valid rego files successfully": {
			files: map[string]string{
				"policy.rego": "package policy\n\nallow := true",
				"data.json":   `{"users": ["alice", "bob"]}`,
				"test.rego":   "package test\n\nimport rego.v1\n\ntest_allow if policy.allow",
			},
			expectChangedFiles: []string{"policy.rego", "test.rego"},
			expectFailedFiles:  []string{},
		},
		"continues processing when some files have parse errors": {
			files: map[string]string{
				"valid.rego":   "package valid\n\nallow := true",
				"invalid.rego": "package invalid\n\n{ invalid syntax here",
				"another.rego": "package another\n\ndeny := false",
			},
			expectChangedFiles: []string{"valid.rego", "invalid.rego", "another.rego"},
			// note that parse errors are business as usual, and errors are written to the file diags
			expectFailedFiles: []string{},
		},
		"handles empty workspace": {
			files:              map[string]string{},
			expectChangedFiles: []string{},
			expectFailedFiles:  []string{},
		},
		"handles file read permission errors gracefully": {
			files: map[string]string{
				"readable.rego":   "package readable\n\nallow := true",
				"unreadable.rego": "package unreadable\n\nallow := true",
			},
			unreadableFiles:    []string{"unreadable.rego"},
			expectChangedFiles: []string{"readable.rego"},
			expectFailedFiles:  []string{"unreadable.rego"},
		},
		"handles multiple file read permission errors": {
			files: map[string]string{
				"readable1.rego":   "package readable1\n\nallow := true",
				"unreadable1.rego": "package unreadable1\n\nallow := true",
				"readable2.rego":   "package readable2\n\ndeny := false",
				"unreadable2.rego": "package unreadable2\n\ndeny := false",
			},
			unreadableFiles:    []string{"unreadable1.rego", "unreadable2.rego"},
			expectChangedFiles: []string{"readable1.rego", "readable2.rego"},
			expectFailedFiles:  []string{"unreadable1.rego", "unreadable2.rego"},
		},
		"processes all files when even with cached files": {
			files: map[string]string{
				"cached1.rego": "package cached1\n\nallow := true",
				"cached2.rego": "package cached2\n\ndeny := false",
				"new.rego":     "package new\n\ncount := 1",
			},
			cachedFiles: []string{"cached1.rego", "cached2.rego"},
			cachedContent: map[string]string{
				"cached1.rego": "package cached1\n\nallow := false",
			},
			// only new.rego & cached1.rego are changed
			expectChangedFiles: []string{"new.rego", "cached1.rego"},
			expectFailedFiles:  []string{},
		},
		"reloads file when disk content differs from cache": {
			files: map[string]string{
				"changed.rego": "package changed\n\nallow := false",
				"same.rego":    "package same\n\ndeny := true",
			},
			cachedFiles: []string{"changed.rego", "same.rego"},
			cachedContent: map[string]string{
				"changed.rego": "package changed\n\nallow := true",
				"same.rego":    "package same\n\ndeny := true",
			},
			expectChangedFiles: []string{"changed.rego"},
			expectFailedFiles:  []string{},
		},
		"ignores changed cached files when newOnly is true": {
			files: map[string]string{
				// disk contents is not matching cache
				"changed-cached.rego": "package changed_cached\n\nallow := false",
				// not cached
				"new.rego": "package new\n\ncount := 1",
			},
			cachedFiles: []string{"changed-cached.rego"},
			cachedContent: map[string]string{
				// cache override
				"changed-cached.rego": "package changed_cached\n\nallow := true",
			},
			newOnly: true,
			// cache-changed is not reported since we asked for newOnly
			expectChangedFiles: []string{"new.rego"},
			expectFailedFiles:  []string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if len(tc.unreadableFiles) > 0 && runtime.GOOS == "windows" {
				t.Skip("file permissions used to test unreadableFiles doesn't work on windows")
			}

			tempDir := testutil.TempDirectoryOf(t, tc.files)

			for _, fileName := range tc.unreadableFiles {
				unreadableFile := filepath.Join(tempDir, fileName)
				must.Equal(t, nil, os.Chmod(unreadableFile, 0o000), "make file %s unreadable %v", fileName)
			}

			opts := &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())}

			server := NewLanguageServer(t.Context(), opts)
			server.workspaceRootURI = "file://" + tempDir
			server.client.Identifier = clients.IdentifierGeneric
			server.loadedConfig = &config.Config{}

			for _, fileName := range tc.cachedFiles {
				var (
					content string
					exists  bool
				)

				if tc.cachedContent != nil {
					content, exists = tc.cachedContent[fileName]
				}

				if !exists {
					content, exists = tc.files[fileName]
					if !exists {
						t.Fatalf("cached file %s not found in test files", fileName)
					}
				}

				fileURI := "file://" + filepath.Join(tempDir, fileName)
				server.cache.SetFileContents(fileURI, content)
			}

			changedURIs, failedFiles, err := server.loadWorkspaceContents(t.Context(), tc.newOnly)
			must.Equal(t, nil, err)

			if len(failedFiles) != len(tc.expectFailedFiles) {
				t.Errorf("expected %d failed files, got %d", len(tc.expectFailedFiles), len(failedFiles))

				for i, f := range failedFiles {
					t.Logf("failed file %d: URI=%s, Error=%v", i, f.URI, f.Error)
				}
			}

			for _, expectedFailedFile := range tc.expectFailedFiles {
				found := false

				for _, failedFile := range failedFiles {
					if filepath.Base(failedFile.URI) == expectedFailedFile {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("expected file %s to be in failed files list", expectedFailedFile)
				}
			}

			if len(tc.expectChangedFiles) > 0 {
				assert.False(t, len(changedURIs) < len(tc.expectChangedFiles), "changed files count")

				for _, expectedFile := range tc.expectChangedFiles {
					found := slices.ContainsFunc(changedURIs, func(changedURI string) bool {
						return filepath.Base(changedURI) == expectedFile
					})

					if !found {
						t.Errorf("expected file %s to be in changed files list", expectedFile)
					}
				}
			}

			for _, expectedFile := range tc.expectChangedFiles {
				for _, changedURI := range changedURIs {
					if filepath.Base(changedURI) == expectedFile {
						assert.True(t, server.cache.HasFileContents(changedURI))

						break
					}
				}
			}
		})
	}
}
