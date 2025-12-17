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

// 66581660 ns/op	36151370 B/op	  937768 allocs/op
// 65395669 ns/op	33885737 B/op	  869142 allocs/op
func BenchmarkCompileBundle(b *testing.B) {
	bndl := bundle.Loaded()
	compiler := NewCompilerWithRegalBuiltins()

	for b.Loop() {
		if compiler.Compile(bndl.ParsedModules("regal")); len(compiler.Errors) > 0 {
			b.Fatal(compiler.Errors)
		}
	}
}
