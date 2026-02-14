package lsp

import (
	"path/filepath"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
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

			result := must.Return(ls.Handle(t.Context(), nil, &jsonrpc2.Request{
				Method: "textDocument/signatureHelp",
				Params: testutil.ToJSONRawMessage(t, types.SignatureHelpParams{
					TextDocument: types.TextDocumentIdentifier{URI: mainRegoURI},
					Position:     tc.position,
				}),
			}))(t)

			signatureHelp := must.Be[*types.SignatureHelp](t, result)

			assert.True(t, len(signatureHelp.Signatures) > 0, "at least one signature")
			assert.DereferenceEqual(t, 0, signatureHelp.ActiveSignature, "activeSignature")
			assert.DereferenceEqual(t, 0, signatureHelp.ActiveParameter, "activeParameter")

			sig := signatureHelp.Signatures[0]

			assert.Equal(t, tc.expectedLabel, sig.Label, "label")
			assert.Equal(t, tc.expectedDoc, sig.Documentation, "documentation")
			assert.Equal(t, len(tc.expectedParams), len(sig.Parameters), "number of parameters")

			for i, expectedParam := range tc.expectedParams {
				assert.Equal(t, expectedParam, sig.Parameters[i].Label, "parameter label")
			}

			assert.DereferenceEqual(t, 0, sig.ActiveParameter, "activeParameter")
		})
	}
}
