package transform

import (
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/io/paths"
	"github.com/open-policy-agent/regal/internal/roast/transforms"
	"github.com/open-policy-agent/regal/internal/roast/transforms/module"
	"github.com/open-policy-agent/regal/pkg/roast/rast"

	_ "github.com/open-policy-agent/regal/internal/encoding/roast"
)

var (
	pathSeparatorTerm              = ast.InternedTerm(paths.Separator)
	environment       [2]*ast.Term = rast.Item("environment", ast.ObjectTerm(
		rast.Item("path_separator", pathSeparatorTerm),
	))

	operationsLintItem        = rast.Item("operations", rast.ArrayTerm("lint"))
	operationsLintCollectItem = rast.Item("operations", rast.ArrayTerm("lint", "collect"))
)

// AnyToValue converts a native Go value x to a Value.
// This is an optimized version of the same function in the OPA codebase,
// and optimized in a way that makes it useful only for a map[string]any
// unmarshaled from RoAST JSON. Don't use it for anything else.
func AnyToValue(x any) (ast.Value, error) {
	return transforms.AnyToValue(x)
}

// ToAST converts a Rego module to an ast.Value suitable for use as input in Regal.
func ToAST(name, content string, mod *ast.Module, collect bool) (ast.Value, error) {
	value, err := module.ToValue(mod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert module to value: %w", err)
	}

	rctx := RegalContextWithOperations(name, content, mod.RegoVersionSpan().String(), collect)

	return insert(value, "regal", ast.NewTerm(rctx))
}

func ToASTWithRegalContext(mod *ast.Module, regalContext ast.Object) (ast.Value, error) {
	value, err := module.ToValue(mod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert module to value: %w", err)
	}

	return insert(value, "regal", ast.NewTerm(regalContext))
}

// RegalContext creates a context object for a Regal input, containing the attributes
// common to most / all Regal use cases.
func RegalContext(name, content, regoVersion string) ast.Object {
	context := ast.NewObject(
		rast.Item("file", ast.ObjectTerm(
			rast.Item("name", ast.StringTerm(name)),
			rast.Item("lines", rast.LinesArrayTerm(content)),
			rast.Item("abs", ast.StringTerm(paths.Abs(name))),
			rast.Item("rego_version", ast.InternedTerm(regoVersion)),
		)),
		environment,
	)

	return context
}

// RegalContextWithOperations creates a Regal context object with operations
// for linting or collecting, depending on the collect parameter.
func RegalContextWithOperations(name, content, regoVersion string, collect bool) ast.Object {
	operations := operationsLintItem
	if collect {
		operations = operationsLintCollectItem
	}

	context := RegalContext(name, content, regoVersion)
	context.Insert(operations[0], operations[1])

	return context
}

func insert(val ast.Value, key string, value *ast.Term) (_ ast.Value, err error) {
	if obj, ok := val.(ast.Object); !ok {
		return nil, fmt.Errorf("can't insert into non-object value: %T", val)
	} else {
		rast.Insert(obj, key, value)
	}

	return val, err
}
