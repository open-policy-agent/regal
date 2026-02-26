package explorer

import (
	"bytes"
	"context"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/open-policy-agent/opa/v1/ir"
	"github.com/open-policy-agent/opa/v1/util"

	regal_compile "github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

type CompileResult struct {
	Stage  ast.StageID
	Result *ast.Module
	Error  string
}

func (cr *CompileResult) FormattedResult() string {
	if cr.Result == nil {
		return ""
	}

	formatted, _ := format.Ast(cr.Result)

	return string(formatted)
}

func CompilerStages(path, rego string, useStrict, useAnno, usePrint bool) []CompileResult {
	c := regal_compile.NewCompilerWithRegalBuiltins().
		WithStrict(useStrict).
		WithEnablePrintStatements(usePrint).
		WithUseTypeCheckAnnotations(useAnno)

	stages := ast.AllStages()
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
		c = c.WithStageAfterID(stage, ast.CompilerStageDefinition{
			Name:       string(stage) + "Record",
			MetricName: string(stage) + "_record",
			Stage: func(c0 *ast.Compiler) *ast.Error {
				result = append(result, CompileResult{Stage: stage, Result: getOne(c0.Modules)})

				return nil
			},
		})
	}

	if c.Compile(map[string]*ast.Module{path: mod}); len(c.Errors) > 0 {
		// stage after the last than ran successfully
		stage := stages[len(result)-1]
		result = append(result, CompileResult{Stage: stage + ": Failure", Error: c.Errors.Error()})
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

	compiler := compile.New().
		WithTarget(compile.TargetPlan).
		WithBundle(&bundle.Bundle{Modules: []bundle.ModuleFile{{URL: "/url", Path: path, Raw: r, Parsed: mod}}}).
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
