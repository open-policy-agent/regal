package regal.lsp.util.find_test

import data.regal.lsp.util.find

test_every_var_at_position_key if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	[var, every_terms] := find.every_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	var.value == "k"
	every_terms.key.value == "k"
	every_terms.value.value == "v"
}

test_every_var_at_position_value if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	[var, _] := find.every_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 10}
		with input.regal.file.lines as split(policy, "\n")

	var.value == "v"
}

test_every_var_at_position_no_match_on_domain if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
	}
}`

	# cursor on 'input' in the domain — not a declared var
	not find.every_var_at_position
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 14}
		with input.regal.file.lines as split(policy, "\n")
}
