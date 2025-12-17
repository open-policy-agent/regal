package transform

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/transforms"
	"github.com/open-policy-agent/regal/internal/roast/transforms/module"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"

	_ "github.com/open-policy-agent/regal/internal/roast/encoding"
)

var (
	pathSeparatorTerm              = ast.InternedTerm(string(os.PathSeparator))
	environment       [2]*ast.Term = ast.Item(ast.InternedTerm("environment"), ast.ObjectTerm(
		ast.Item(ast.InternedTerm("path_separator"), pathSeparatorTerm),
	))

	operationsLintItem        = ast.Item(ast.InternedTerm("operations"), ast.ArrayTerm(ast.InternedTerm("lint")))
	operationsLintCollectItem = ast.Item(ast.InternedTerm("operations"), ast.ArrayTerm(
		ast.InternedTerm("lint"),
		ast.InternedTerm("collect")),
	)
)

// AnyToValue converts a native Go value x to a Value.
// This is an optimized version of the same function in the OPA codebase,
// and optimized in a way that makes it useful only for a map[string]any
// unmarshaled from RoAST JSON. Don't use it for anything else.
func AnyToValue(x any) (ast.Value, error) {
	return transforms.AnyToValue(x)
}

// ToOPAInputValue converts provided x to an ast.Value suitable for use as
// parsed input to OPA (`rego.EvalParsedInput`). This will have the value
// pass through the same kind of roundtrip as OPA would otherwise have to
// do when provided unparsed input, but much more efficiently as both JSON
// marshalling and the custom InterfaceToValue function provided here are
// optimized for performance.
func ToOPAInputValue(x any) (ast.Value, error) {
	ptr := reference(x)
	if err := anyPtrRoundTrip(ptr); err != nil {
		return nil, err
	}

	return AnyToValue(*ptr)
}

// ToAST converts a Rego module to an ast.Value suitable for use as input in Regal.
func ToAST(name, content string, mod *ast.Module, collect bool) (ast.Value, error) {
	value, err := module.ToValue(mod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert module to value: %w", err)
	}

	//nolint:forcetypeassert
	value.(ast.Object).Insert(ast.InternedTerm("regal"), ast.NewTerm(
		RegalContextWithOperations(name, content, mod.RegoVersion().String(), collect),
	))

	return value, nil
}

// RegalContext creates a context object for a Regal input, containing the attributes
// common to most / all Regal use cases.
func RegalContext(name, content, regoVersion string) ast.Object {
	abs, _ := filepath.Abs(name)

	context := ast.NewObject(
		ast.Item(ast.InternedTerm("file"), ast.ObjectTerm(
			ast.Item(ast.InternedTerm("name"), ast.StringTerm(name)),
			ast.Item(ast.InternedTerm("lines"), rast.LinesArrayTerm(content)),
			ast.Item(ast.InternedTerm("abs"), ast.StringTerm(abs)),
			ast.Item(ast.InternedTerm("rego_version"), ast.InternedTerm(regoVersion)),
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

// From OPA's util package
//
// Reference returns a pointer to its argument unless the argument already is
// a pointer. If the argument is **t, or ***t, etc, it will return *t.
//
// Used for preparing Go types (including pointers to structs) into values to be
// put through util.RoundTrip().
func reference(x any) *any {
	var y any

	rv := reflect.ValueOf(x)
	if rv.Kind() == reflect.Ptr {
		return reference(rv.Elem().Interface())
	}

	if rv.Kind() != reflect.Invalid {
		y = rv.Interface()

		return &y
	}

	return &x
}

func anyPtrRoundTrip(x *any) error {
	bs, err := jsoniter.ConfigFastest.Marshal(x)
	if err != nil {
		return err
	}

	if err = jsoniter.ConfigFastest.Unmarshal(bs, x); err != nil {
		return encoding.SafeNumberConfig.Unmarshal(bs, x)
	}

	return nil
}
