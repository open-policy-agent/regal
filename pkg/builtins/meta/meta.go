package meta

import (
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
)

var (
	// RegalParseModule metadata for regal.parse_module.
	RegalParseModule = &rego.Function{
		Name: "regal.parse_module",
		Decl: types.NewFunction(
			types.Args(
				types.Named("filename", types.S).Description("file name to attach to AST nodes' locations"),
				types.Named("rego", types.S).Description("Rego module"),
			),
			types.Named("output", types.NewObject(nil, types.NewDynamicProperty(types.S, types.A))),
		),
	}

	// RegalLast metadata for regal.last.
	RegalLast = &rego.Function{
		Name: "regal.last",
		Decl: types.NewFunction(
			types.Args(
				types.Named("array", types.NewArray(nil, types.A)).
					Description("performance optimized last index retrieval"),
			),
			types.Named("element", types.A),
		),
	}

	// RegalIsFormatted metadata for regal.is_formatted.
	RegalIsFormatted = &rego.Function{
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

	RegalParseModuleBuiltin = &ast.Builtin{Name: RegalParseModule.Name, Decl: RegalParseModule.Decl, CanSkipBctx: true}
	RegalLastBuiltin        = &ast.Builtin{Name: RegalLast.Name, Decl: RegalLast.Decl, CanSkipBctx: true}
	RegalIsFormattedBuiltin = &ast.Builtin{Name: RegalIsFormatted.Name, Decl: RegalIsFormatted.Decl, CanSkipBctx: true}
)
