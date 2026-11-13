package compile

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/metrics"

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
// 70071275 ns/op	37088672 B/op	  767943 allocs/op
// 57304216 ns/op	33593914 B/op	  708960 allocs/op
// 56471829 ns/op	33621029 B/op	  635587 allocs/op // no stages skipped
// 42137205 ns/op	25306061 B/op	  466088 allocs/op // skipping stages
func BenchmarkCompileBundle(b *testing.B) {
	bndl := bundle.Loaded()

	b.Run("all stages", func(b *testing.B) {
		for b.Loop() {
			compiler := NewCompilerWithRegalBuiltins()
			mustCompile(b, compiler, bndl.ParsedModules("regal"))
		}
	})

	b.Run("all stages with schemas", func(b *testing.B) {
		for b.Loop() {
			compiler := NewCompilerWithRegalBuiltins().
				WithSchemas(RegalSchemaSet()).
				WithUseTypeCheckAnnotations(true)

			mustCompile(b, compiler, bndl.ParsedModules("regal"))
		}
	})

	b.Run("skip stages safe for embedded rego", func(b *testing.B) {
		for b.Loop() {
			compiler := NewCompilerWithRegalBuiltins().WithSkipStages(EmbeddedRegoSkipStages[:]...)
			mustCompile(b, compiler, bndl.ParsedModules("regal"))
		}
	})
}

func TestCompileWithSchemas(t *testing.T) {
	t.Parallel()

	compiler := NewCompilerWithRegalBuiltins().WithSchemas(RegalSchemaSet())
	mustCompile(t, compiler, bundle.Loaded().ParsedModules("regal"))
}

func TestCompileWithMetrics(t *testing.T) {
	t.Parallel()

	m := metrics.New()
	c := NewCompilerWithRegalBuiltins().
		WithMetrics(m).
		WithStrict(true).
		WithSchemas(RegalSchemaSet()).
		WithUseTypeCheckAnnotations(true).
		WithSkipStages(EmbeddedRegoSkipStages[:]...)

	mustCompile(t, c, bundle.Loaded().ParsedModules("regal"))

	// Uncomment and paste the outputJSON into a file with formatting
	// to see the wallclock time for each of the stages. Useful!
	//
	// t.Log(string(must.Return(json.Marshal(m))(t)))
	// t.Fail()
}

func TestAllExpectedStagesRun(t *testing.T) {
	t.Parallel()

	s := ast.AllStages()
	c := NewCompilerWithRegalBuiltins()
	n := 0
	f := func(*ast.Compiler) *ast.Error {
		n++
		return nil
	}

	for _, stage := range s {
		c = c.WithStageAfterID(stage, ast.CompilerStageDefinition{Stage: f})
	}

	mustCompile(t, c, bundle.Loaded().ParsedModules("regal"))

	if n != len(s) {
		t.Fatalf("expected all %d stages to run, but only %d ran", len(s), n)
	}
}

func mustCompile(tb testing.TB, c *ast.Compiler, mods map[string]*ast.Module) {
	tb.Helper()

	if c.Compile(mods); len(c.Errors) > 0 {
		tb.Fatal(c.Errors.Error())
	}
}
