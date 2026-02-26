package regal.lsp.semantictokens.vars.imports_test

import data.regal.lsp.semantictokens.vars.imports

test_imports if {
	policy := `package regal.woo

import data.regal.ast

test_function(param1, param2) := result if {
	ast.is_constant
}`
	tokens := imports.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "3:19:3:22", "type": "string", "value": "ast"} in tokens
}
