package regal.lsp.util.find_test

import data.regal.lsp.util.find

test_some_var_at_position_idx if {
	policy := `package p

myrule := r if {
	some idx, user in input.users
	r := user.name
}`

	module := regal.parse_module("p.rego", policy)
	[var, rule_index] := find.some_var_at_position
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.regal as module.regal

	var.value == "idx"
	rule_index == 0
}

test_some_var_at_position_user if {
	policy := `package p

myrule := r if {
	some idx, user in input.users
	r := user.name
}`

	module := regal.parse_module("p.rego", policy)
	[var, rule_index] := find.some_var_at_position
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 11}
		with input.regal as module.regal

	var.value == "user"
	rule_index == 0
}

test_some_var_at_position_no_match_on_keyword if {
	policy := `package p

myrule := r if {
	some idx, user in input.users
	r := user.name
}`

	module := regal.parse_module("p.rego", policy)
	not find.some_var_at_position
		with data.workspace.parsed["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 1}
		with input.regal as module.regal
}
