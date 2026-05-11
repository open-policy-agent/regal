package rego_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

type document struct {
	uri     string
	content string
	parsed  map[string]any
}

func newDocument(uri, content string) document {
	return document{
		uri:     uri,
		content: content,
		parsed:  encoding.MustJSONRoundTripTo[map[string]any](parse.MustParseModule(content)),
	}
}

func TestRouteTextDocumentCodeAction(t *testing.T) {
	t.Parallel()

	mgr := rego.NewRegoRouter(t.Context(), nil, query.NewCache(), providers(regalContext(), "", ""))
	req := request("textDocument/codeAction", codeActionParams(t, "file:///workspace/p.rego", 0, 0, 0, 10))
	rsp := must.Return(mgr.Handle(t.Context(), nil, req))(t)

	must.Be[*json.RawMessage](t, rsp)
}

func TestRouteTextDocumentDocumentLink(t *testing.T) {
	t.Parallel()

	doc := newDocument("file:///workspace/p.rego", "# regal ignore:prefer-snake-case\npackage p\n")
	stg := inmem.NewFromObjectWithOpts(map[string]any{"workspace": map[string]any{
		"parsed": map[string]any{doc.uri: doc.parsed},
		"config": map[string]any{
			"rules": map[string]any{
				"style": map[string]any{"prefer-snake-case": map[string]any{}},
			},
		},
	}}, inmem.OptRoundTripOnWrite(false))

	rct := &rego.RegalContext{
		File: rego.File{
			Name:  "workspace/p.rego",
			Lines: []string{"# regal ignore:prefer-snake-case", "package p"},
		},
	}
	mgr := rego.NewRegoRouter(t.Context(), stg, query.NewCache(), providers(rct, "", ""))
	rsp := must.Return(mgr.Handle(t.Context(), nil, request("textDocument/documentLink", linkParams(t, doc.uri))))(t)

	must.Be[*json.RawMessage](t, rsp)
}

func TestRouteTextDocumentDocumentHighlight(t *testing.T) {
	t.Parallel()

	doc := newDocument("file:///workspace/p.rego", "# METADATA\n# title: p\npackage p\n")
	stg := inmem.NewFromObjectWithOpts(map[string]any{"workspace": map[string]any{
		"parsed": map[string]any{doc.uri: doc.parsed},
	}}, inmem.OptRoundTripOnWrite(false))
	mgr := rego.NewRegoRouter(t.Context(), stg, query.NewCache(), rego.Providers{
		ContextProvider: func(string, *rego.Requirements) *rego.RegalContext {
			return regalContext()
		},
		ContentProvider: func(uri string) (string, bool) {
			return doc.content, uri == doc.uri
		},
	})
	prm := docPositionParams(t, doc.uri, types.Position{Line: 0, Character: 4})
	rsp := must.Return(mgr.Handle(t.Context(), nil, request("textDocument/documentHighlight", prm)))(t)

	must.Be[*json.RawMessage](t, rsp)
}

func TestRouteIgnoredDocument(t *testing.T) {
	t.Parallel()

	mgr := rego.NewRegoRouter(
		t.Context(), nil, query.NewCache(), providers(regalContext(), "", "file:///workspace/ignored.rego"),
	)
	req := request("textDocument/codeAction", codeActionParams(t, "file:///workspace/ignored.rego", 0, 0, 0, 10))
	rsp := must.Return(mgr.Handle(t.Context(), nil, req))(t)

	must.Equal(t, nil, rsp, "response for ignored document")
}

func TestTextDocumentSignatureHelp(t *testing.T) {
	t.Parallel()

	doc := newDocument("file:///workspace/p.rego", `package example

allow if regex.match(`+"`foo`"+`, "bar")
allow if count([1,2,3]) == 2
allow if concat(",", "a", "b") == "b,a"`)

	store := inmem.NewFromObjectWithOpts(map[string]any{"workspace": map[string]any{
		"builtins": map[string]any{
			"count":       ast.Count,
			"concat":      ast.Concat,
			"regex.match": ast.RegexMatch,
		},
		"parsed": map[string]any{doc.uri: doc.parsed},
	}}, inmem.OptRoundTripOnWrite(false))

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

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			mgr := rego.NewRegoRouter(ctx, store, query.NewCache(), providers(regalContext(), doc.content, ""))
			req := request("textDocument/signatureHelp", docPositionParams(t, doc.uri, tc.position))
			rsp := must.Return(mgr.Handle(ctx, nil, req))(t)

			signatureHelp := must.Be[*types.SignatureHelp](t, rsp)
			assert.True(t, len(signatureHelp.Signatures) > 0, "expected at least one signature")
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

func TestRouteCompletionItemResolve(t *testing.T) {
	t.Parallel()

	store := inmem.NewFromObjectWithOpts(map[string]any{"workspace": map[string]any{
		"builtins": map[string]any{"count": ast.Count},
	}}, inmem.OptReturnASTValuesOnRead(true))

	mgr := rego.NewRegoRouter(t.Context(), store, query.NewCache(), providers(regalContext(), "", ""))
	req := request("completionItem/resolve", testutil.ToJSONRawMessage(t, map[string]any{
		"label": "count",
		"data":  map[string]any{"resolver": "builtins"},
	}))
	ret := must.Return(mgr.Handle(t.Context(), nil, req))(t)
	cmi := must.Be[types.CompletionItem](t, ret)

	must.NotEqual(t, nil, cmi.Documentation, "documentation is set")
	must.Equal(t, "markdown", cmi.Documentation.Kind, "documentation kind")
}

func TestRouteInitialize(t *testing.T) {
	t.Parallel()

	data := map[string]any{"server": map[string]any{"version": "0.2.0"}}
	store := inmem.NewFromObjectWithOpts(data, inmem.OptReturnASTValuesOnRead(true))

	mgr := rego.NewRegoRouter(t.Context(), store, query.NewCache(), rego.Providers{})
	mgr.RegisterResultHandler("initialize", func(_ context.Context, result any) (any, error) {
		rsp, ok := result.(rego.InitializeResponse)

		must.Equal(t, true, ok, "initialize response", rsp)
		must.Equal(t, "Regal", rsp.Response.ServerInfo.Name, "server name")
		must.Equal(t, "0.2.0", rsp.Response.ServerInfo.Version, "server version")
		must.Equal(t, true, rsp.Regal.Client.InitOptions.EnableExplorer, "client supports explorer")

		return rsp, nil
	})

	req := request("initialize", testutil.ToJSONRawMessage(t, map[string]any{
		"processId":             12345,
		"rootUri":               "file:///workspace",
		"clientInfo":            map[string]any{"name": "Visual Studio Code"},
		"initializationOptions": map[string]any{"enableExplorer": true},
	}))
	rsp := must.Return(mgr.Handle(t.Context(), nil, req))(t)

	must.Be[rego.InitializeResponse](t, rsp)
}

func docPositionParams(t *testing.T, uri string, position types.Position) *json.RawMessage {
	t.Helper()

	return testutil.ToJSONRawMessage(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     position,
	})
}

func codeActionParams(t *testing.T, uri string, ls, cs, le, ce int) *json.RawMessage {
	t.Helper()

	return testutil.ToJSONRawMessage(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]int{"line": ls, "character": cs},
			"end":   map[string]int{"line": le, "character": ce},
		},
		"context": map[string]any{
			"diagnostics": []map[string]any{{
				"code":    "opa-fmt",
				"message": "Format using opa-fmt",
				"range": map[string]any{
					"start": map[string]int{"line": ls, "character": cs},
					"end":   map[string]int{"line": le, "character": ce},
				},
			}},
		},
	})
}

func linkParams(t *testing.T, uri string) *json.RawMessage {
	t.Helper()

	return testutil.ToJSONRawMessage(t, map[string]any{"textDocument": map[string]any{"uri": uri}})
}

func providers(rc *rego.RegalContext, content, ignored string) rego.Providers {
	return rego.Providers{
		ContextProvider: func(string, *rego.Requirements) *rego.RegalContext {
			return rc
		},
		IgnoredProvider: func(uri string) bool {
			return uri == ignored
		},
		ContentProvider: func(_ string) (string, bool) {
			return content, content != ""
		},
	}
}

func regalContext() *rego.RegalContext {
	return &rego.RegalContext{
		Environment: rego.Environment{
			PathSeparator:    "/",
			WorkspaceRootURI: "file:///workspace",
			WebServerBaseURI: "http://webserver",
		},
		File: rego.File{Name: "workspace/p.rego", Abs: "/workspace/p.rego"},
	}
}

func request(method string, params *json.RawMessage) *jsonrpc2.Request {
	return &jsonrpc2.Request{Method: method, Params: params}
}
