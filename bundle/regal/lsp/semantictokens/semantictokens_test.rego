package regal.lsp.semantictokens_test

import data.regal.lsp.semantictokens

test_result if {
	policy := `package regal.woo

import data.regal.ast

test_function(param1, param2) := result if {
	ast.is_constant
}`
	module := regal.parse_module("p.rego", policy)
	result := semantictokens.result with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	result == {"response": {
		"imports": {
			{"col": 0, "length": 6, "line": 2, "type": 3},
			{"col": 18, "length": 3, "line": 2, "type": 2},
		},
		"packages": {
			{"col": 0, "length": 7, "line": 0, "modifiers": 0, "type": 3},
			{"col": 14, "length": 3, "line": 0, "modifiers": 0, "type": 0},
		},
		"vars": {
			{"col": 14, "length": 6, "line": 4, "modifiers": 1, "type": 1},
			{"col": 22, "length": 6, "line": 4, "modifiers": 1, "type": 1},
		},
	}}
}
