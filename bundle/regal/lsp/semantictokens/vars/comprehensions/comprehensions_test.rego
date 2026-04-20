package regal.lsp.semantictokens.vars.comprehensions_test

import data.regal.lsp.semantictokens.vars.comprehensions

test_array_comprehension if {
	policy := `package regal.woo

array_comprehensions := [x |
	some i, x in [1, 2, 3]
	i == 2
]`

	tokens := comprehensions.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 9, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 25, "length": 1, "line": 2, "modifiers": 2, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 2, "type": 1},
	}
}

test_set_comprehension if {
	policy := `package regal.woo

set_comprehensions := {x |
	some i, x in [1, 2, 3]
	i == 2
}`

	tokens := comprehensions.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 9, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 23, "length": 1, "line": 2, "modifiers": 2, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 2, "type": 1},
	}
}

test_object_comprehension if {
	policy := `package regal.woo

object_comprehensions := {k: v |
	some k, v in [1, 2, 3]
	v == 2
}`

	tokens := comprehensions.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 6, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 9, "length": 1, "line": 3, "modifiers": 1, "type": 1},
		{"col": 26, "length": 1, "line": 2, "modifiers": 2, "type": 1},
		{"col": 29, "length": 1, "line": 2, "modifiers": 2, "type": 1},
		{"col": 1, "length": 1, "line": 4, "modifiers": 2, "type": 1},
	}
}
