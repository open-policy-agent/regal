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

// 67207511 ns/op  42652069 B/op    1032387 allocs/op
// 62502125 ns/op  35918354 B/op     929634 allocs/op
func BenchmarkCompileBundle(b *testing.B) {
	bndl := bundle.Loaded()
	compiler := NewCompilerWithRegalBuiltins()

	for b.Loop() {
		if compiler.Compile(bndl.ParsedModules("regal")); len(compiler.Errors) > 0 {
			b.Fatal(compiler.Errors)
		}
	}
}
