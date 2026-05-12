package rego_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/semantictokens"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

// This mirrors TestRegoHandlers 1:1, and no validation of result is done here.
// Therefore make sure all tests in TestRegoHandlers pass before running these benchmarks!
// See testdata/bench_results.txt for the current results, and update it on changes.
func BenchmarkRegoHandlers(b *testing.B) {
	for name, test := range handlerTests(b) { //nolint:gocritic // "each iteration copies 184 bytes"
		b.Run(name, func(b *testing.B) {
			stg := storeForDocument(b, test.policy, test.data.content)
			mgr := rego.NewRegoRouter(b.Context(), stg, query.NewCache(), providersForTest(b, test))
			mgr.RegisterResultHandler("textDocument/semanticTokens/full", semantictokens.ResultHandler)

			runBenchmark(b, mgr, request(test.method, new(json.RawMessage(test.input.content))))
		})
	}
}

// 214834 ns/op	  116999 B/op	    3187 allocs/op
// 224618 ns/op	  122690 B/op	    3292 allocs/op
func BenchmarkFoldingRangeHandlerLineFoldingOnlyVsFull(b *testing.B) {
	doc := newDocument("file:///p.rego", must.ReadFile(b, "testdata/foldingranges.rego"))
	req := request("textDocument/foldingRange", testutil.ToJSONRawMessage(b, map[string]any{
		"textDocument": map[string]any{"uri": doc.uri},
	}))

	for name, lineFoldingOnly := range map[string]bool{
		"lineFoldingOnly=false": false,
		"lineFoldingOnly=true":  true,
	} {
		b.Run(name, func(b *testing.B) {
			store := inmem.NewFromObjectWithOpts(map[string]any{
				"client": map[string]any{"capabilities": map[string]any{"textDocument": map[string]any{
					"foldingRange": map[string]any{
						"lineFoldingOnly": lineFoldingOnly,
					},
				}}},
				"workspace": map[string]any{
					"parsed": map[string]any{doc.uri: doc.parsed},
				},
			}, inmem.OptReturnASTValuesOnRead(true))
			router := rego.NewRegoRouter(b.Context(), store, query.NewCache(), contextForDoc(doc))

			runBenchmark(b, router, req)
		})
	}
}

func runBenchmark(b *testing.B, mgr *rego.RegoRouter, req *jsonrpc2.Request) {
	b.Helper()

	for b.Loop() {
		if _, err := mgr.Handle(b.Context(), nil, req); err != nil {
			b.Fatal(err)
		}
	}
}

func contextForDoc(doc document) rego.Providers {
	return rego.Providers{
		ContextProvider: func(string, rego.Requirements) *rego.RegalContext {
			return &rego.RegalContext{
				File: rego.File{Lines: strings.Split(doc.content, "\n")},
			}
		},
	}
}
