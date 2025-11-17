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
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/builtins/meta"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	RegalBuiltinRegoFuncs = []func(*rego.Rego){
		rego.Function1(meta.RegalLast, RegalLast),
		rego.Function2(meta.RegalParseModule, RegalParseModule),
		rego.Function2(meta.RegalIsFormatted, RegalIsFormatted),
	}

	regoVersionTerm = ast.InternedTerm("rego_version")
)

// RegalParseModule regal.parse_module, like rego.parse_module but with location data included in AST.
func RegalParseModule(_ rego.BuiltinContext, filename, policy *ast.Term) (*ast.Term, error) {
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

func RegalIsFormatted(_ rego.BuiltinContext, input, options *ast.Term) (*ast.Term, error) {
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
	source := util.StringToByteSlice(inputStr)

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

	return result, err
}

// TestContextBuiltins returns the list of builtins as expected by the test runner.
func TestContextBuiltins() []*tester.Builtin {
	return []*tester.Builtin{
		{
			Decl: &ast.Builtin{Name: meta.RegalParseModule.Name, Decl: meta.RegalParseModule.Decl},
			Func: rego.Function2(meta.RegalParseModule, RegalParseModule),
		},
		{
			Decl: &ast.Builtin{Name: meta.RegalLast.Name, Decl: meta.RegalLast.Decl},
			Func: rego.Function1(meta.RegalLast, RegalLast),
		},
		{
			Decl: &ast.Builtin{Name: meta.RegalIsFormatted.Name, Decl: meta.RegalIsFormatted.Decl},
			Func: rego.Function2(meta.RegalIsFormatted, RegalIsFormatted),
		},
	}
}
