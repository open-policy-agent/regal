package compile

import (
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/embeds"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

var (
	// EmbeddedRegoSkipStages lists the stages we may skip when compiling Regal's embedded modules if:
	// - They are the *only* modules being compiled, i.e. no custom, user-provided modules.
	// - The modules are compiled in a released binary, not one being actively developed.
	EmbeddedRegoSkipStages = [...]ast.StageID{
		ast.StageCheckTypes,
		ast.StageCheckRecursion,
		ast.StageRewritePrintCalls,
		ast.StageCheckRuleConflicts,
		ast.StageCheckUnusedImports,
		ast.StageCheckUndefinedFuncs,
		ast.StageCheckUnsafeBuiltins,
		ast.StageCheckSafetyRuleHeads,
		ast.StageCheckKeywordOverrides,
		ast.StageCheckDuplicateImports,
		ast.StageCheckDeprecatedBuiltins,
		ast.StageRewriteTestRulesForTracing,
	}

	// RegalSchemaSet returns a SchemaSet containing the Regal schemas embedded in the binary.
	// Currently only used by the test command. Should we want to expand the use of this later,
	// we'll probably want to only read the schemas relevant to the context.
	RegalSchemaSet = sync.OnceValue(func() *ast.SchemaSet {
		schemaSet, _ := files.NewWalker("schemas").
			WithFilter(filter.JSON).
			WithFileSystem(embeds.SchemasFS).
			Reduce(ast.NewSchemaSet(), func(path string, acc *ast.SchemaSet) (*ast.SchemaSet, error) {
				bs := util.Must(embeds.SchemasFS.ReadFile(path))
				schemaAny := util.Must(encoding.JSONUnmarshalTo[any](bs))

				// > This is unlike io/fs.WalkDir, which always uses slash separated paths.
				// https://pkg.go.dev/path/filepath#WalkDir
				//
				// https://github.com/open-policy-agent/regal/issues/1679
				spl := strings.Split(strings.TrimSuffix(path, ".json"), "/")
				ref := ast.Ref([]*ast.Term{ast.SchemaRootDocument}).Extend(ast.MustParseRef(strings.Join(spl[1:], ".")))

				acc.Put(ref, schemaAny)

				return acc, nil
			})

		return schemaSet
	})
)

func NewCompilerWithRegalBuiltins() *ast.Compiler {
	return ast.NewCompiler().WithCapabilities(io.Capabilities())
}
