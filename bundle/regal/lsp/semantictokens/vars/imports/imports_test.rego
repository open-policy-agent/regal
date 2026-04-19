package regal.lsp.semantictokens.vars.imports_test

import data.regal.lsp.semantictokens.vars.imports

test_imports if {
	policy := `package regal.woo

import data.regal.ast`

	module := regal.parse_module("p.rego", policy)

	tokens := imports.result with input.params as {"textDocument": {"uri": "file://p.rego"}}
		with input.regal as module.regal
		with data.workspace.parsed["file://p.rego"] as module

	tokens == {
		{"col": 0, "length": 6, "line": 2, "type": 3},
		{"col": 18, "length": 3, "line": 2, "type": 2},
	}
}

test_imports_with_alias if {
	policy := `package regal.woo

import data.regal.ast
import data.other.identifier as alias`

	module := regal.parse_module("p.rego", policy)

	tokens := imports.result with input.params as {"textDocument": {"uri": "file://p.rego"}}
		with input.regal as module.regal
		with data.workspace.parsed["file://p.rego"] as module

	tokens == {
		{"col": 0, "length": 6, "line": 2, "type": 3},
		{"col": 18, "length": 3, "line": 2, "type": 2},
		{"col": 0, "length": 6, "line": 3, "type": 3},
		{"col": 32, "length": 5, "line": 3, "type": 2},
	}
}
