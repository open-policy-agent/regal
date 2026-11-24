package regal.lsp.semantictokens.vars.some_expr_test

import data.regal.lsp.semantictokens.vars.some_expr

test_some_two_vars if {
	policy := `package regal.woo

some_two_vars if {
	some i, item in input.array
	i < 10
	item > 0
}`

	module := regal.parse_module("p.rego", policy)
	tokens := some_expr.result
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal as module.regal

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 2, "type": 1},
		{"col": 9, "length": 4, "line": 3, "modifiers": 2, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 4, "type": 1},
		{"col": 1, "length": 4, "line": 5, "modifiers": 4, "type": 1},
	}
}

test_some_single_var if {
	policy := `package regal.woo

some_one_var if {
	some i in input.array
	i < 10
}`

	module := regal.parse_module("p.rego", policy)
	tokens := some_expr.result
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal as module.regal

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 2, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 4, "type": 1},
	}
}
