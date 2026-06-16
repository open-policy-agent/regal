package regal.lsp.rename_test

import data.regal.lsp.rename

test_rename_arg_cross_file if {
	policy_a := `package shared

greet(name) := msg if {
	msg := sprintf("hello, %s", [name])
}`
	policy_b := `package shared

greet(name) := msg if {
	count(name) > 50
	msg := sprintf("hi %s", [name])
}`

	resp := rename.result.response with data.workspace.parsed as {
		"file:///a.rego": regal.parse_module("a.rego", policy_a),
		"file:///b.rego": regal.parse_module("b.rego", policy_b),
	}
		with input.params.textDocument.uri as "file:///a.rego"
		with input.params.position as {"line": 2, "character": 6}
		with input.params.newName as "user"
		with input.regal.file.lines as split(policy_a, "\n")

	resp == {"changes": {
		"file:///a.rego": {
			{"newText": "user", "range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}}},
			{"newText": "user", "range": {"start": {"line": 3, "character": 30}, "end": {"line": 3, "character": 34}}},
		},
		"file:///b.rego": {
			{"newText": "user", "range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}}},
			{"newText": "user", "range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 11}}},
			{"newText": "user", "range": {"start": {"line": 4, "character": 26}, "end": {"line": 4, "character": 30}}},
		},
	}}
}

test_rename_some_var_single_file if {
	policy := `package p

list_admins contains admin if {
	some user in input.users
	user.role == "admin"
	admin := user.name
}`

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.params.newName as "u"
		with input.regal.file.lines as split(policy, "\n")

	resp == {"changes": {"file:///p.rego": {
		{"newText": "u", "range": {"start": {"line": 3, "character": 6}, "end": {"line": 3, "character": 10}}},
		{"newText": "u", "range": {"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 5}}},
		{"newText": "u", "range": {"start": {"line": 5, "character": 10}, "end": {"line": 5, "character": 14}}},
	}}}
}

test_rename_no_match if {
	policy := `package p

foo := 42`

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 11}
		with input.params.newName as "x"
		with input.regal.file.lines as split(policy, "\n")

	resp == null
}
