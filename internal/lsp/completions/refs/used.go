package refs

import (
	"context"
	"fmt"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/rego"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/pkg/builtins"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"

	_ "embed"
)

var (
	refNamesQuery = rast.RefStringToBody(`data.regal.lsp.completion.ref_names`)
	pqOnce        = sync.OnceValues(prepareQuery)
)

// initialize prepares the rego query for finding ref names used in a module.
// This is run and the resulting prepared query stored for performance reasons.
// This function is only used by language server code paths and so init() is not
// used.
func prepareQuery() (*rego.PreparedEvalQuery, error) {
	dataBundle := bundle.Bundle{
		Manifest: bundle.Manifest{
			Roots:    &[]string{"internal"},
			Metadata: map[string]any{"name": "internal"},
		},
		Data: map[string]any{
			"internal": map[string]any{
				"combined_config": map[string]any{
					"capabilities": rio.ToMap(config.CapabilitiesForThisVersion()),
				},
			},
		},
	}

	regoArgs := append([]func(*rego.Rego){
		rego.ParsedBundle("regal", rbundle.LoadedBundle()),
		rego.ParsedBundle("internal", &dataBundle),
		rego.ParsedQuery(refNamesQuery),
	}, builtins.RegalBuiltinRegoFuncs...)

	preparedQuery, err := rego.New(regoArgs...).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	return &preparedQuery, nil
}

// UsedInModule returns a list of ref names suitable for completion that are
// used in the module's code.
// See the rego above for more details on what's included and excluded.
// This function is run when the parse completes for a module.
func UsedInModule(ctx context.Context, module *ast.Module) ([]string, error) {
	inputValue, err := transform.ModuleToValue(module)
	if err != nil {
		return nil, fmt.Errorf("failed converting input to value: %w", err)
	}

	pq, err := pqOnce()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare rego query: %w", err)
	}

	rs, err := pq.Eval(ctx, rego.EvalParsedInput(inputValue))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate rego query: %w", err)
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		// no refs found
		return []string{}, nil
	}

	foundRefs, ok := rs[0].Expressions[0].Value.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", rs[0].Expressions[0].Value)
	}

	refNames := make([]string, len(foundRefs))
	for i, ref := range foundRefs {
		refNames[i], ok = ref.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T", ref)
		}
	}

	return refNames, nil
}
