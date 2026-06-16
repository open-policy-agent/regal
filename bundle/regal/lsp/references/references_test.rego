package regal.lsp.references_test

import data.regal.lsp.references

test_references_arg_cross_file if {
	policy_a := `package shared

greet(name) := msg if {
	msg := sprintf("hello, %s", [name])
}`
	policy_b := `package shared

greet(name) := msg if {
	count(name) > 50
	msg := sprintf("hi %s", [name])
}`

	refs := references.result.response with data.workspace.parsed as {
		"file:///a.rego": regal.parse_module("a.rego", policy_a),
		"file:///b.rego": regal.parse_module("b.rego", policy_b),
	}
		with input.params.textDocument.uri as "file:///a.rego"
		with input.params.position as {"line": 2, "character": 6}
		with input.regal.file.lines as split(policy_a, "\n")

	refs == {
		{"uri": "file:///a.rego", "range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}}},
		{"uri": "file:///a.rego", "range": {"start": {"line": 3, "character": 30}, "end": {"line": 3, "character": 34}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 11}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 4, "character": 26}, "end": {"line": 4, "character": 30}}},
	}
}

test_references_some_var_single_rule if {
	policy := `package p

list_admins contains admin if {
	some user in input.users
	user.role == "admin"
	admin := user.name
}

other_rule contains x if {
	some user in input.items
	x := user.label
}`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 3, "character": 6}, "end": {"line": 3, "character": 10}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 5}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 5, "character": 10}, "end": {"line": 5, "character": 14}}},
	}
}

test_references_no_match if {
	policy := `package p

foo := 42`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 11}
		with input.regal.file.lines as split(policy, "\n")

	refs == set()
}
