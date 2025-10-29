package lsp

import (
	"path/filepath"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestTextDocumentSignatureHelp(t *testing.T) {
	t.Parallel()

	mainRegoURI := uri.FromPath(clients.IdentifierGoTest, filepath.Join(t.TempDir(), "main.rego"))
	content := `package example

allow if regex.match(` + "`foo`" + `, "bar")
allow if count([1,2,3]) == 2
allow if concat(",", "a", "b") == "b,a"`

	ls := NewLanguageServer(t.Context(), &LanguageServerOptions{Logger: log.NewLogger(log.LevelDebug, t.Output())})
	ls.cache.SetFileContents(mainRegoURI, content)

	testCases := map[string]struct {
		position       types.Position
		expectedLabel  string
		expectedDoc    string
		expectedParams []string
	}{
		"regex.match function call": {
			position:       types.Position{Line: 2, Character: 21},
			expectedLabel:  "regex.match(pattern: string, value: string) -> boolean",
			expectedDoc:    "Matches a string against a regular expression.",
			expectedParams: []string{"pattern: string", "value: string"},
		},
		"count function call": {
			position:       types.Position{Line: 3, Character: 16},
			expectedLabel:  "count(collection: any) -> number",
			expectedDoc:    "Count takes a collection or string and returns the number of elements (or characters) in it.",
			expectedParams: []string{"collection: any"},
		},
		"concat function call": {
			position:       types.Position{Line: 4, Character: 17},
			expectedLabel:  "concat(delimiter: string, collection: any) -> string",
			expectedDoc:    "Joins a set or array of strings with a delimiter.",
			expectedParams: []string{"delimiter: string", "collection: any"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := testutil.Must(ls.Handle(t.Context(), nil, &jsonrpc2.Request{
				Method: "textDocument/signatureHelp",
				Params: testutil.ToJSONRawMessage(t, types.SignatureHelpParams{
					TextDocument: types.TextDocumentIdentifier{URI: mainRegoURI},
					Position:     tc.position,
				}),
			}))(t)

			signatureHelp := testutil.MustBe[*types.SignatureHelp](t, result)

			if len(signatureHelp.Signatures) == 0 {
				t.Error("expected at least one signature")
			}

			if signatureHelp.ActiveSignature == nil {
				t.Error("expected ActiveSignature to be set")
			} else if *signatureHelp.ActiveSignature != 0 {
				t.Errorf("expected ActiveSignature to be 0, got %d", *signatureHelp.ActiveSignature)
			}

			if signatureHelp.ActiveParameter == nil {
				t.Error("expected ActiveParameter to be set")
			} else if *signatureHelp.ActiveParameter != 0 {
				t.Errorf("expected ActiveParameter to be 0, got %d", *signatureHelp.ActiveParameter)
			}

			sig := signatureHelp.Signatures[0]

			if sig.Label != tc.expectedLabel {
				t.Errorf("expected signature label to be '%s', got '%s'", tc.expectedLabel, sig.Label)
			}

			if sig.Documentation != tc.expectedDoc {
				t.Errorf("expected documentation to be '%s', got '%s'", tc.expectedDoc, sig.Documentation)
			}

			if len(sig.Parameters) != len(tc.expectedParams) {
				t.Fatalf("expected %d parameters, got %d", len(tc.expectedParams), len(sig.Parameters))
			}

			for i, expectedParam := range tc.expectedParams {
				if sig.Parameters[i].Label != expectedParam {
					t.Errorf("expected parameter %d label to be '%s', got '%s'", i, expectedParam, sig.Parameters[i].Label)
				}
			}

			if sig.ActiveParameter == nil {
				t.Error("expected signature ActiveParameter to be set")
			} else if *sig.ActiveParameter != 0 {
				t.Errorf("expected signature ActiveParameter to be 0, got %d", *sig.ActiveParameter)
			}
		})
	}
}
