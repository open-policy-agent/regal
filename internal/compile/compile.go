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

func NewCompilerWithRegalBuiltins() *ast.Compiler {
	return ast.NewCompiler().WithCapabilities(io.Capabilities())
}

// RegalSchemaSet returns a SchemaSet containing the Regal schemas embedded in the binary.
// Currently only used by the test command. Should we want to expand the use of this later,
// we'll probably want to only read the schemas relevant to the context.
var RegalSchemaSet = sync.OnceValue(func() *ast.SchemaSet {
	schemaSet, _ := files.DefaultWalkReducer("schemas", ast.NewSchemaSet()).
		WithFilters(filter.Not(filter.Suffixes(".json"))).
		ReduceFS(embeds.SchemasFS, func(path string, schemaSet *ast.SchemaSet) (*ast.SchemaSet, error) {
			schemaAny := util.Must(encoding.JSONUnmarshalTo[any](util.Must(embeds.SchemasFS.ReadFile(path))))

			// > This is unlike io/fs.WalkDir, which always uses slash separated paths.
			// https://pkg.go.dev/path/filepath#WalkDir
			//
			// https://github.com/open-policy-agent/regal/issues/1679
			spl := strings.Split(strings.TrimSuffix(path, ".json"), "/")
			ref := ast.Ref([]*ast.Term{ast.SchemaRootDocument}).Extend(ast.MustParseRef(strings.Join(spl[1:], ".")))

			schemaSet.Put(ref, schemaAny)

			return schemaSet, nil
		})

	return schemaSet
})
