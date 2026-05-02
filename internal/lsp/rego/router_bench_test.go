package rego_test

import (
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// 214834 ns/op	  116999 B/op	    3187 allocs/op
// 224618 ns/op	  122690 B/op	    3292 allocs/op
func BenchmarkFoldingRangeHandler(b *testing.B) {
	doc := newDocument("file:///p.rego", must.ReadFile(b, "testdata/foldingranges.rego"))
	req := request("textDocument/foldingRange", testutil.ToJSONRawMessage(b, map[string]any{
		"textDocument": map[string]any{"uri": doc.uri},
	}))

	for name, lineFoldingOnly := range map[string]bool{
		"lineFoldingOnly=false": false,
		"lineFoldingOnly=true":  true,
	} {
		b.Run(name, func(b *testing.B) {
			router := rego.NewRegoRouter(b.Context(), store(doc, lineFoldingOnly), query.NewCache(), emptyContext())
			runBenchmark(b, router, req)
		})
	}
}

// 68920 ns/op	   56301 B/op	    1123 allocs/op
// 58037 ns/op	   48892 B/op	    1002 allocs/op - using 'undecoded' handler
// 55144 ns/op	   46362 B/op	     977 allocs/op - using 'undecoded' handler and custom Value decoder
// 53628 ns/op	   43976 B/op	     791 allocs/op - using 'undecoded' handler and custom Value encoder
func BenchmarkCodeActionHandler(b *testing.B) {
	doc := newDocument("file:///p.rego", "package ignored")
	req := request("textDocument/codeAction", testutil.ToJSONRawMessage(b, map[string]any{
		"textDocument": map[string]any{"uri": doc.uri},
		"context": map[string]any{
			"diagnostics": []map[string]any{{
				"code":    "use-assignment-operator",
				"message": "foobar",
				"range": map[string]any{
					"start": map[string]any{"line": 2, "character": 4},
					"end":   map[string]any{"line": 2, "character": 10},
				},
				"source": "regal/style",
			}},
		},
	}))

	router := rego.NewRegoRouter(b.Context(), store(doc, false), query.NewCache(), emptyContext())
	runBenchmark(b, router, req)
}

func runBenchmark(b *testing.B, mgr *rego.RegoRouter, req *jsonrpc2.Request) {
	b.Helper()

	for b.Loop() {
		if _, err := mgr.Handle(b.Context(), nil, req); err != nil {
			b.Fatal(err)
		}
	}
}

func store(doc document, lineFoldingOnly bool) storage.Store {
	return inmem.NewFromObjectWithOpts(map[string]any{
		"client": map[string]any{"capabilities": map[string]any{"textDocument": map[string]any{
			"foldingRange": map[string]any{
				"lineFoldingOnly": lineFoldingOnly,
			},
		}}},
		"workspace": map[string]any{
			"parsed": map[string]any{doc.uri: doc.parsed},
		},
	}, inmem.OptReturnASTValuesOnRead(true))
}

func emptyContext() rego.Providers {
	return rego.Providers{ContextProvider: func(string, *rego.Requirements) *rego.RegalContext {
		return &rego.RegalContext{}
	}}
}
