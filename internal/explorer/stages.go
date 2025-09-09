package explorer

import (
	"bytes"
	"context"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
	"github.com/open-policy-agent/opa/v1/ir"
	"github.com/open-policy-agent/opa/v1/util"

	compile2 "github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

type CompileResult struct {
	Stage  string
	Result *ast.Module
	Error  string
}

type stage struct{ name, metricName string }

// NOTE(sr): copied from 0.68.0.
var stages = []stage{
	{"ResolveRefs", "compile_stage_resolve_refs"},
	{"InitLocalVarGen", "compile_stage_init_local_var_gen"},
	{"RewriteRuleHeadRefs", "compile_stage_rewrite_rule_head_refs"},
	{"CheckKeywordOverrides", "compile_stage_check_keyword_overrides"},
	{"CheckDuplicateImports", "compile_stage_check_duplicate_imports"},
	{"RemoveImports", "compile_stage_remove_imports"},
	{"SetModuleTree", "compile_stage_set_module_tree"},
	{"SetRuleTree", "compile_stage_set_rule_tree"},
	{"RewriteLocalVars", "compile_stage_rewrite_local_vars"},
	{"CheckVoidCalls", "compile_stage_check_void_calls"},
	{"RewritePrintCalls", "compile_stage_rewrite_print_calls"},
	{"RewriteExprTerms", "compile_stage_rewrite_expr_terms"},
	{"ParseMetadataBlocks", "compile_stage_parse_metadata_blocks"},
	{"SetAnnotationSet", "compile_stage_set_annotationset"},
	{"RewriteRegoMetadataCalls", "compile_stage_rewrite_rego_metadata_calls"},
	{"SetGraph", "compile_stage_set_graph"},
	{"RewriteComprehensionTerms", "compile_stage_rewrite_comprehension_terms"},
	{"RewriteRefsInHead", "compile_stage_rewrite_refs_in_head"},
	{"RewriteWithValues", "compile_stage_rewrite_with_values"},
	{"CheckRuleConflicts", "compile_stage_check_rule_conflicts"},
	{"CheckUndefinedFuncs", "compile_stage_check_undefined_funcs"},
	{"CheckSafetyRuleHeads", "compile_stage_check_safety_rule_heads"},
	{"CheckSafetyRuleBodies", "compile_stage_check_safety_rule_bodies"},
	{"RewriteEquals", "compile_stage_rewrite_equals"},
	{"RewriteDynamicTerms", "compile_stage_rewrite_dynamic_terms"},
	{"RewriteTestRulesForTracing", "compile_stage_rewrite_test_rules_for_tracing"}, // must run after RewriteDynamicTerms
	{"CheckRecursion", "compile_stage_check_recursion"},
	{"CheckTypes", "compile_stage_check_types"},
	{"CheckUnsafeBuiltins", "compile_state_check_unsafe_builtins"},
	{"CheckDeprecatedBuiltins", "compile_state_check_deprecated_builtins"},
	{"BuildRuleIndices", "compile_stage_rebuild_indices"},
	{"BuildComprehensionIndices", "compile_stage_rebuild_comprehension_indices"},
	{"BuildRequiredCapabilities", "compile_stage_build_required_capabilities"},
}

func CompilerStages(path, rego string, useStrict, useAnno, usePrint bool) []CompileResult {
	c := compile2.NewCompilerWithRegalBuiltins().
		WithStrict(useStrict).
		WithEnablePrintStatements(usePrint).
		WithUseTypeCheckAnnotations(useAnno)

	result := append(make([]CompileResult, 0, len(stages)+1), CompileResult{Stage: "ParseModule"})
	opts := parse.ParserOptions()
	opts.ProcessAnnotation = useAnno

	mod, err := ast.ParseModuleWithOpts(path, rego, opts)
	if err != nil {
		result[0].Error = err.Error()

		return result
	}

	result[0].Result = mod

	for i := range stages {
		stage := stages[i]
		c = c.WithStageAfter(stage.name,
			ast.CompilerStageDefinition{
				Name:       stage.name + "Record",
				MetricName: stage.metricName + "_record",
				Stage: func(c0 *ast.Compiler) *ast.Error {
					result = append(result, CompileResult{
						Stage:  stage.name,
						Result: getOne(c0.Modules),
					})

					return nil
				},
			})
	}

	c.Compile(map[string]*ast.Module{
		path: mod,
	})

	if len(c.Errors) > 0 {
		// stage after the last than ran successfully
		stage := stages[len(result)-1]
		result = append(result, CompileResult{
			Stage: stage.name + ": Failure",
			Error: c.Errors.Error(),
		})
	}

	return result
}

func getOne(mods map[string]*ast.Module) *ast.Module {
	for _, m := range mods {
		return m.Copy()
	}

	panic("unreachable")
}

func Plan(ctx context.Context, path, rego string, usePrint bool) (string, error) {
	mod, err := ast.ParseModuleWithOpts(path, rego, parse.ParserOptions())
	if err != nil {
		return "", err
	}

	r := util.StringToByteSlice(rego)
	b := &bundle.Bundle{Modules: []bundle.ModuleFile{{URL: "/url", Path: path, Raw: r, Parsed: mod}}}

	compiler := compile.New().
		WithTarget(compile.TargetPlan).
		WithBundle(b).
		WithRegoAnnotationEntrypoints(true).
		WithEnablePrintStatements(usePrint)
	if err := compiler.Build(ctx); err != nil {
		return "", err
	}

	policy, err := encoding.JSONUnmarshalTo[ir.Policy](compiler.Bundle().PlanModules[0].Raw)
	if err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	if err := ir.Pretty(&buf, &policy); err != nil {
		return "", err
	}

	return buf.String(), nil
}
