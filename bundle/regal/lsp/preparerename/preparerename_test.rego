package regal.lsp.preparerename_test

import data.regal.lsp.preparerename

test_preparerename_arg if {
	policy := `package p

greet(name) := msg if {
	msg := sprintf("hello, %s", [name])
}`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 6}
		with input.regal.file.lines as split(policy, "\n")

	resp == {
		"placeholder": "name",
		"range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}},
	}
}

test_preparerename_some_var if {
	policy := `package p

list_admins contains admin if {
	some user in input.users
	user.role == "admin"
	admin := user.name
}`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.regal.file.lines as split(policy, "\n")

	resp == {
		"placeholder": "user",
		"range": {"start": {"line": 3, "character": 6}, "end": {"line": 3, "character": 10}},
	}
}

test_preparerename_no_match if {
	policy := `package p

simple := 42`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 11}
		with input.regal.file.lines as split(policy, "\n")

	resp == null
}

test_preparerename_every_var if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	resp == {
		"placeholder": "k",
		"range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 8}},
	}
}

test_preparerename_comprehension_var if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
		x > 0
	]
}`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	resp == {
		"placeholder": "x",
		"range": {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 8}},
	}
}
