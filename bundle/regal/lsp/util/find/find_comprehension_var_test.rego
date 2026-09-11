package regal.lsp.util.find_test

import data.regal.lsp.util.find

test_comprehension_var_at_position_array if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
		x > 0
	]
}`

	[var, comp] := find.comprehension_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	var.value == "x"
	comp.type == "arraycomprehension"
}

test_comprehension_var_at_position_object if {
	policy := `package p

r := result if {
	result := {k: v |
		some k, v in input.items
	}
}`

	[var, comp] := find.comprehension_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	var.value == "k"
	comp.type == "objectcomprehension"
}

test_comprehension_var_at_position_no_match_on_output if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
	]
}`

	# cursor on the output 'x' (line 3, col 12) — that's a reference not a declaration
	not find.comprehension_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 12}
		with input.regal.file.lines as split(policy, "\n")
}
