package builtins

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/tester"
	"github.com/open-policy-agent/opa/v1/topdown/builtins"
	"github.com/open-policy-agent/opa/v1/types"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

// RegalParseModuleMeta metadata for regal.parse_module.
var RegalParseModuleMeta = &rego.Function{
	Name: "regal.parse_module",
	Decl: types.NewFunction(
		types.Args(
			types.Named("filename", types.S).Description("file name to attach to AST nodes' locations"),
			types.Named("rego", types.S).Description("Rego module"),
		),
		types.Named("output", types.NewObject(nil, types.NewDynamicProperty(types.S, types.A))),
	),
}

// RegalLastMeta metadata for regal.last.
var RegalLastMeta = &rego.Function{
	Name: "regal.last",
	Decl: types.NewFunction(
		types.Args(
			types.Named("array", types.NewArray(nil, types.A)).
				Description("performance optimized last index retrieval"),
		),
		types.Named("element", types.A),
	),
}

var RegalBuiltinRegoFuncs = []func(*rego.Rego){
	rego.Function1(RegalLastMeta, RegalLast),
	rego.Function2(RegalParseModuleMeta, RegalParseModule),
	rego.Function2(RegalIsFormattedMeta, RegalIsFormatted),
}

// RegalIsFormattedMeta metadata for regal.is_formatted.
var RegalIsFormattedMeta = &rego.Function{
	Name: "regal.is_formatted",
	Decl: types.NewFunction(
		types.Args(
			types.Named("input", types.S).
				Description("input string to check for formatting"),
			types.Named("options", types.NewObject(nil, types.NewDynamicProperty(types.S, types.A))).
				Description("formatting options"),
		),
		types.B,
	),
}

// RegalParseModule regal.parse_module, like rego.parse_module but with location data included in AST.
func RegalParseModule(_ rego.BuiltinContext, filename *ast.Term, policy *ast.Term) (*ast.Term, error) {
	filenameValue, err := builtins.StringOperand(filename.Value, 1)
	if err != nil {
		return nil, err
	}

	policyValue, err := builtins.StringOperand(policy.Value, 2)
	if err != nil {
		return nil, err
	}

	filenameStr := string(filenameValue)
	policyStr := string(policyValue)

	opts := parse.ParserOptions()

	// Allow testing Rego v0 modules. We could provide a separate builtin for this,
	// but the need for this will likely diminish over time, so let's start simple.
	if strings.HasSuffix(filenameStr, "_v0.rego") {
		opts.RegoVersion = ast.RegoV0
	}

	mod, err := ast.ParseModuleWithOpts(filenameStr, policyStr, opts)
	if err != nil {
		return nil, err
	}

	roast, err := transform.ToAST(filenameStr, policyStr, mod, false)
	if err != nil {
		return nil, err
	}

	return ast.NewTerm(roast), nil
}

// RegalLast regal.last returns the last element of an array.
func RegalLast(_ rego.BuiltinContext, arr *ast.Term) (*ast.Term, error) {
	arrOp, err := builtins.ArrayOperand(arr.Value, 1)
	if err != nil {
		return nil, err
	}

	if arrOp.Len() > 0 {
		return arrOp.Elem(arrOp.Len() - 1), nil
	}

	// index out of bounds, but returning an error allocates
	// and we have no use for this information anyway.
	return nil, nil //nolint:nilnil
}

var regoVersionTerm = ast.StringTerm("rego_version")

func RegalIsFormatted(_ rego.BuiltinContext, input *ast.Term, options *ast.Term) (*ast.Term, error) {
	inputStr, err := builtins.StringOperand(input.Value, 1)
	if err != nil {
		return nil, err
	}

	optionsObj, err := builtins.ObjectOperand(options.Value, 2)
	if err != nil {
		return nil, err
	}

	regoVersion := ast.RegoV1
	if versionTerm := optionsObj.Get(regoVersionTerm); versionTerm != nil {
		if v, ok := versionTerm.Value.(ast.String); ok && v == "v0" {
			regoVersion = ast.RegoV0
		}
	}

	// We don't need to process annotations for formatting.
	popts := ast.ParserOptions{ProcessAnnotation: false, RegoVersion: regoVersion}
	source := util.StringToByteSlice(string(inputStr))

	result, err := formatRego(source, format.Opts{RegoVersion: regoVersion, ParserOptions: &popts})
	if err != nil {
		return nil, err
	}

	return ast.InternedTerm(bytes.Equal(source, result)), nil
}

func formatRego(source []byte, opts format.Opts) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case string:
				err = fmt.Errorf("error formatting: %s", r)
			case error:
				err = r
			default:
				err = fmt.Errorf("error formatting: %v", r)
			}
		}
	}()

	result, err = format.SourceWithOpts("", source, opts)

	return
}

// TestContextBuiltins returns the list of builtins as expected by the test runner.
func TestContextBuiltins() []*tester.Builtin {
	return []*tester.Builtin{
		{
			Decl: &ast.Builtin{Name: RegalParseModuleMeta.Name, Decl: RegalParseModuleMeta.Decl},
			Func: rego.Function2(RegalParseModuleMeta, RegalParseModule),
		},
		{
			Decl: &ast.Builtin{Name: RegalLastMeta.Name, Decl: RegalLastMeta.Decl},
			Func: rego.Function1(RegalLastMeta, RegalLast),
		},
		{
			Decl: &ast.Builtin{Name: RegalIsFormattedMeta.Name, Decl: RegalIsFormattedMeta.Decl},
			Func: rego.Function2(RegalIsFormattedMeta, RegalIsFormatted),
		},
	}
}
