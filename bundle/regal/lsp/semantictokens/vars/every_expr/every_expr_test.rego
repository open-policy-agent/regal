package regal.lsp.semantictokens.vars.every_expr_test

import data.regal.lsp.semantictokens.vars.every_expr

test_every_two_vars if {
	policy := `package regal.woo

every_two_vars if {
	every k, v in input.object {
		is_string(k)
		v > 0
	}
}`

	tokens := every_expr.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 7, "length": 1, "line": 3, "modifiers": 2, "type": 1},
		{"col": 10, "length": 1, "line": 3, "modifiers": 2, "type": 1},
		{"col": 12, "length": 1, "line": 4, "modifiers": 4, "type": 1},
		{"col": 2, "length": 1, "line": 5, "modifiers": 4, "type": 1},
	}
}

test_every_single_var if {
	policy := `package regal.woo

every_one_var if {
	every k in input.object {
		is_string(k)
	}
}`

	tokens := every_expr.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 7, "length": 1, "line": 3, "modifiers": 2, "type": 1},
		{"col": 12, "length": 1, "line": 4, "modifiers": 4, "type": 1},
	}
}
