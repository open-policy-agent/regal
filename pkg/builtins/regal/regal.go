package regal

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
	"github.com/open-policy-agent/opa/v1/topdown/builtins"
	"github.com/open-policy-agent/opa/v1/types"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	// ParseModule metadata for regal.parse_module.
	ParseModule = &ast.Builtin{
		Name:        "regal.parse_module",
		Description: "Parses a Regal module similarly to rego.parse_module, but returns a RoAST representation.",
		Decl: types.NewFunction(
			types.Args(
				types.Named("filename", types.S).Description("file name to attach to AST nodes' locations"),
				types.Named("rego", types.S).Description("Rego module"),
			),
			types.Named("output", types.NewObject(nil, types.NewDynamicProperty(types.S, types.A))),
		),
		CanSkipBctx: true,
	}

	// Last metadata for regal.last.
	Last = &ast.Builtin{
		Name:        "regal.last",
		Description: "Function optimized for retrieving the last element of an array.",
		Decl: types.NewFunction(
			types.Args(
				types.Named("array", types.NewArray(nil, types.A)).
					Description("performance optimized last index retrieval"),
			),
			types.Named("element", types.A),
		),
		CanSkipBctx: true,
	}

	// IsFormatted metadata for regal.is_formatted.
	IsFormatted = &ast.Builtin{
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
		CanSkipBctx: true,
	}
)

func init() {
	ast.RegisterBuiltin(ParseModule)
	ast.RegisterBuiltin(Last)
	ast.RegisterBuiltin(IsFormatted)

	topdown.RegisterBuiltinFunc(ParseModule.Name, RegalParseModule)
	topdown.RegisterBuiltinFunc(Last.Name, RegalLast)
	topdown.RegisterBuiltinFunc(IsFormatted.Name, RegalIsFormatted)
}

// RegalParseModule regal.parse_module, like rego.parse_module but with location data included in AST.
func RegalParseModule(_ rego.BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	filenameValue, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	policyValue, err := builtins.StringOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	filenameStr := string(filenameValue)
	policyStr := string(policyValue)

	opts := ast.ParserOptions{ProcessAnnotation: true}

	// Allow testing Rego v0 modules. We could provide a separate builtin for this,
	// but the need for this will likely diminish over time, so let's start simple.
	if strings.HasSuffix(filenameStr, "_v0.rego") {
		opts.RegoVersion = ast.RegoV0
	}

	mod, err := ast.ParseModuleWithOpts(filenameStr, policyStr, opts)
	if err != nil {
		return err
	}

	roast, err := transform.ToAST(filenameStr, policyStr, mod, false)
	if err != nil {
		return err
	}

	return iter(ast.NewTerm(roast))
}

// RegalLast regal.last returns the last element of an array.
func RegalLast(_ rego.BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	arrOp, err := builtins.ArrayOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	if arrOp.Len() > 0 {
		return iter(arrOp.Elem(arrOp.Len() - 1))
	}

	// index out of bounds, but have no use for this information anyway
	return nil
}

func RegalIsFormatted(_ rego.BuiltinContext, operands []*ast.Term, iter func(*ast.Term) error) error {
	inputStr, err := builtins.StringOperand(operands[0].Value, 1)
	if err != nil {
		return err
	}

	optionsObj, err := builtins.ObjectOperand(operands[1].Value, 2)
	if err != nil {
		return err
	}

	regoVersion := ast.RegoV1

	if versionTerm := optionsObj.Get(ast.InternedTerm("rego_version")); versionTerm != nil {
		if v, ok := versionTerm.Value.(ast.String); ok && v == "v0" {
			regoVersion = ast.RegoV0
		}
	}

	// We don't need to process annotations for formatting.
	popts := ast.ParserOptions{ProcessAnnotation: false, RegoVersion: regoVersion}
	source := util.StringToByteSlice(inputStr)

	result, err := formatRego(source, format.Opts{RegoVersion: regoVersion, ParserOptions: &popts})
	if err != nil {
		return err
	}

	return iter(ast.InternedTerm(bytes.Equal(source, result)))
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
