package compile

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/bundle"
)

func TestSchemaSet(t *testing.T) {
	t.Parallel()

	if RegalSchemaSet().Get(ast.SchemaRootRef.Extend(ast.MustParseRef("regal.lsp.codeaction"))) == nil {
		t.Fatal("expected regal.lsp.codeaction schema to be present in RegalSchemaSet")
	}
}

// 16	  66555594 ns/op	50239492 B/op	 1083664 allocs/op - main
// 18	  62569440 ns/op	38723015 B/op	  944277 allocs/op - compiler-optimizations pr
func BenchmarkCompileBundle(b *testing.B) {
	bndl := bundle.LoadedBundle()
	compiler := NewCompilerWithRegalBuiltins()

	for b.Loop() {
		if compiler.Compile(bndl.ParsedModules("regal")); len(compiler.Errors) > 0 {
			b.Fatal(compiler.Errors)
		}
	}
}
