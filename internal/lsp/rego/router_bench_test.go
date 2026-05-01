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

// 227279 ns/op  131120 B/op    3466 allocs/op
// 217259 ns/op  125051 B/op    3340 allocs/op
func BenchmarkFoldingRangeHandler(b *testing.B) {
	doc := newDocument("file:///p.rego", must.ReadFile(b, "testdata/foldingranges.rego"))

	req := request("textDocument/foldingRange", testutil.ToJSONRawMessage(b, map[string]any{
		"textDocument": map[string]any{
			"uri": doc.uri,
		},
	}))

	b.Run("lineFoldingOnly = false", func(b *testing.B) {
		runBenchmark(b, rego.NewRegoRouter(b.Context(), store(doc, false), query.NewCache(), emptyContext()), req)
	})

	b.Run("lineFoldingOnly = true", func(b *testing.B) {
		runBenchmark(b, rego.NewRegoRouter(b.Context(), store(doc, true), query.NewCache(), emptyContext()), req)
	})
}

func runBenchmark(b *testing.B, mgr *rego.RegoRouter, req *jsonrpc2.Request) {
	b.Helper()

	var rsp any

	for b.Loop() {
		var err error
		if rsp, err = mgr.Handle(b.Context(), nil, req); err != nil {
			b.Fatal(err)
		}
	}

	if ranges := must.Be[[]any](b, rsp); len(ranges) != 8 {
		b.Fatalf("expected 8 folding ranges, got %d", len(ranges))
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
