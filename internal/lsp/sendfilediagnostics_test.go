package lsp

import (
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/test"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

// TestSendFileDiagnosticsEmptyArrays replicates the scenario from
// https://github.com/open-policy-agent/regal/issues/1609 where a file that's been
// deleted from the cache has null rather than empty arrays as diagnostics.
func TestSendFileDiagnosticsEmptyArrays(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parseErrors         []types.Diagnostic
		lintErrors          []types.Diagnostic
		fileInCache         bool
		expectedDiagnostics []types.Diagnostic
	}{
		"lint errors only": {
			parseErrors:         []types.Diagnostic{},
			lintErrors:          []types.Diagnostic{{Message: "lint error"}},
			fileInCache:         true,
			expectedDiagnostics: []types.Diagnostic{{Message: "lint error"}},
		},
		"parse errors only": {
			parseErrors:         []types.Diagnostic{{Message: "parse error"}},
			lintErrors:          []types.Diagnostic{},
			fileInCache:         true,
			expectedDiagnostics: []types.Diagnostic{{Message: "parse error"}},
		},
		"both empty in cache": {
			parseErrors:         []types.Diagnostic{},
			lintErrors:          []types.Diagnostic{},
			fileInCache:         true,
			expectedDiagnostics: []types.Diagnostic{},
		},
		"file deleted from cache": {
			// file deleted, and so nothing in the cache
			fileInCache:         false,
			expectedDiagnostics: []types.Diagnostic{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			receivedDiagnostics := make(chan types.FileDiagnostics, 1)
			clientHandler := test.HandlerFor(methodTdPublishDiagnostics, test.SendsToChannel(receivedDiagnostics))

			fileURI := "file:///test.rego"
			ls, _ := createAndInitServer(t, t.Context(), t.TempDir(), clientHandler)

			if tc.fileInCache {
				ls.cache.SetParseErrors(fileURI, tc.parseErrors)
				ls.cache.SetFileDiagnostics(fileURI, tc.lintErrors)
			}

			ls.sendFileDiagnostics(t.Context(), fileURI)

			select {
			case diag := <-receivedDiagnostics:
				if diag.URI != fileURI {
					t.Fatalf("expected URI %s, got %s", fileURI, diag.URI)
				}

				if len(tc.expectedDiagnostics) == 0 && diag.Items == nil {
					t.Errorf("expected empty array [], got nil")
				}

				if len(diag.Items) != len(tc.expectedDiagnostics) {
					t.Errorf("expected %d diagnostics, got %d", len(tc.expectedDiagnostics), len(diag.Items))
				}

				for i, expected := range tc.expectedDiagnostics {
					if i < len(diag.Items) && diag.Items[i].Message != expected.Message {
						t.Errorf("diagnostic %d: expected message %s, got %s", i, expected.Message, diag.Items[i].Message)
					}
				}

			case <-time.After(100 * time.Millisecond):
				t.Fatal("no diagnostics received before timeout")
			}
		})
	}
}
