package regal.lsp.semantictokens.vars.some_expr_test

import data.regal.lsp.semantictokens.vars.some_expr

test_some_two_vars if {
	policy := `package regal.woo

some_two_vars if {
	some i, item in input.array
	i < 10
	item > 0
}`

	tokens := some_expr.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 9, "length": 4, "line": 3, "modifiers": 1, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 2, "type": 1},
		{"col": 1, "length": 4, "line": 5, "modifiers": 2, "type": 1},
	}
}

test_some_single_var if {
	policy := `package regal.woo

some_one_var if {
	some i in input.array
	i < 10
}`

	tokens := some_expr.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 2, "type": 1},
	}
}
