package rego_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mdast "github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/semantictokens"
	"github.com/open-policy-agent/regal/internal/lsp/store"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/roast/transforms/module"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type (
	document struct {
		uri     string
		content string
		parsed  ast.Value
	}
	jsonData struct {
		name    string
		content []byte
	}
	testCase struct {
		method string
		policy document
		input  jsonData
		output jsonData
		data   jsonData
	}
)

func TestRegoHandlers(t *testing.T) {
	t.Parallel()

	testsExecuted := 0

	for name, test := range handlerTests(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Query cache can't be shared due to synchronization issues in our concurrent
			// map implementation... to be fixed later, but only a problem in tests, as the
			// language server doesn't launch more than one Rego router.
			stg := storeForDocument(t, test.policy, test.data.content)
			mgr := rego.NewRegoRouter(t.Context(), stg, query.NewCache(), providersForTest(t, test))
			mgr.RegisterResultHandler("textDocument/semanticTokens/full", semantictokens.ResultHandler)

			req := request(test.method, new(json.RawMessage(test.input.content)))
			rsp := must.Return(mgr.Handle(t.Context(), nil, req))(t)

			// Round-trip needed to ensure difference isn't merely formatting and order
			got := jsonRoundTrip(t, *must.Be[*json.RawMessage](t, rsp))
			exp := jsonRoundTrip(t, test.output.content)

			if !bytes.Equal(exp, got) {
				t.Errorf("expected: %s\ngot: %s", toJSONPretty(t, exp), toJSONPretty(t, got))
			}
		})

		testsExecuted++
	}

	if testsExecuted == 0 {
		t.Fatal("no tests found or executed!")
	}
}

func newDocument(uri, content string) document {
	roast, err := module.ToValue(parse.MustParseModule(content))
	if err != nil {
		panic(err)
	}

	return document{uri: uri, content: content, parsed: roast}
}

func markdownToTest(tb testing.TB, method, src string) testCase {
	tb.Helper()

	md := parser.New().Parse([]byte(must.ReadFile(tb, src)))
	tc := testCase{method: method}

	var currentHeading string

	mdast.WalkFunc(md, func(node mdast.Node, entering bool) mdast.WalkStatus {
		if entering {
			switch n := node.(type) {
			case *mdast.Heading:
				if n.Level == 4 && len(n.Children) == 1 {
					currentHeading = string(n.Container.Children[0].AsLeaf().Literal)
				}
			case *mdast.CodeBlock:
				if currentHeading == "input.json" { //nolint:gocritic
					tc.input = jsonData{name: "input.json", content: n.Literal}
				} else if currentHeading == "data.json" {
					tc.data = jsonData{name: "data.json", content: n.Literal}
				} else if currentHeading == "output.json" {
					tc.output = jsonData{name: "output.json", content: n.Literal}
				} else if strings.HasSuffix(currentHeading, ".rego") {
					tc.policy = newDocument("file:///workspace/"+currentHeading, string(n.Literal))
				}

				currentHeading = ""
			}
		}

		return mdast.GoToNext
	})

	return tc
}

func jsonRoundTrip(t *testing.T, data []byte) []byte {
	t.Helper()

	var m any
	must.Equal(t, nil, json.Unmarshal(data, &m))

	return must.Return(json.Marshal(m))(t)
}

func toJSONPretty(t *testing.T, data []byte) string {
	t.Helper()

	buf := new(bytes.Buffer)
	must.Equal(t, nil, json.Indent(buf, data, "", "  "))

	return buf.String()
}

func storeForDocument(tb testing.TB, doc document, jsonData []byte) storage.Store {
	tb.Helper()

	data := ast.NewObject(rast.Item("workspace", ast.ObjectTerm()))
	if doc.uri != "" && doc.parsed != nil {
		workspace, _ := rast.GetValue[ast.Object](data, "workspace")
		rast.Insert(workspace, "parsed", ast.ObjectTerm(rast.Item(doc.uri, ast.NewTerm(doc.parsed))))
	}

	if len(jsonData) > 0 {
		value := must.Return(encoding.OfValue().Decode(jsonData))(tb)
		if obj, ok := value.(ast.Object); !ok {
			tb.Fatalf("expected JSON data to decode to an object, got %T", value)
		} else if data, ok = data.Merge(obj); !ok {
			tb.Fatalf("failed to merge JSON data into existing data")
		}
	}

	stg := inmem.NewWithOpts(inmem.OptReturnASTValuesOnRead(true))
	if err := storage.WriteOne(tb.Context(), stg, storage.AddOp, storage.RootPath, data); err != nil {
		tb.Fatalf("failed to write to store: %v", err)
	}

	bis := rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())
	must.Equal(tb, nil, store.PutBuiltins(tb.Context(), stg, bis), "failed to update builtins in storage")

	return stg
}

func handlerTests(tb testing.TB) iter.Seq2[string, testCase] {
	tb.Helper()

	return func(yield func(string, testCase) bool) {
		dir := filepath.Join("testdata", "router")
		for _, pattern := range []string{"**/**/*.md", "**/**/**/*.md"} {
			gfs := must.Return(fs.Glob(os.DirFS(dir), pattern))(tb)
			for _, path := range gfs {
				test := markdownToTest(tb, filepath.Dir(path), filepath.Join(dir, path))
				if !yield(path, test) {
					return
				}
			}
		}
	}
}

func providersForTest(tb testing.TB, test testCase) rego.Providers {
	tb.Helper()

	return rego.Providers{
		ContextProvider: func(string, rego.Requirements) *rego.RegalContext {
			return &rego.RegalContext{
				Environment: rego.Environment{WorkspaceRootURI: "file:///workspace", PathSeparator: "/"},
				File:        rego.File{Abs: "/workspace/p.rego"},
			}
		},
		ContentProvider: func(uri string) (string, bool) {
			return test.policy.content, uri == test.policy.uri
		},
		ParseErrorsProvider: func(string) ([]types.Diagnostic, bool) {
			return nil, false
		},
		SuccessfulParseCountProvider: func(string) (uint, bool) {
			return 1, true
		},
	}
}

func TestRouteIgnoredDocument(t *testing.T) {
	t.Parallel()

	uri := "file:///workspace/ignored.rego"
	mgr := rego.NewRegoRouter(t.Context(), nil, query.NewCache(), rego.Providers{IgnoredProvider: func(string) bool {
		return true
	}})
	req := request("textDocument/signatureHelp", docPositionParams(t, uri, types.Position{Line: 0, Character: 0}))
	rsp, err := mgr.Handle(t.Context(), nil, req)

	must.Equal(t, nil, rsp, "no response for ignored document")
	must.Equal(t, nil, err, "no error for ignored document")
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

func request(method string, params *json.RawMessage) *jsonrpc2.Request {
	return &jsonrpc2.Request{Method: method, Params: params}
}
