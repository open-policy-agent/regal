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
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}}
}

test_preparerename_arg_from_use if {
	policy := `package p

greet(name) := msg if {
	msg := sprintf("hello, %s", [name])
}`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 31}
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 3, "character": 30}, "end": {"line": 3, "character": 34}}
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
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 3, "character": 6}, "end": {"line": 3, "character": 10}}
}

test_preparerename_no_match if {
	policy := `package p

simple := 42`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 11}
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/prepareRename"

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
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 8}}
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
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 8}}
}

test_preparerename_import_not_supported[name] if {
	policy := `package p

import data.pkgb as wow

allow if wow.b == 1`

	some name, position in {"path": [2, 10], "alias": [2, 21], "use": [4, 10]}

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": position[0], "character": position[1]}
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/prepareRename"

	resp == null
}

test_preparerename_assign_var[name] if {
	policy := `package p

allow if {
	foo := 1
	foo == 1
}`

	some name, position in {
		"declaration": {"line": 3, "character": 2},
		"use": {"line": 4, "character": 2},
	}

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as position
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": position.line, "character": 1}, "end": {"line": position.line, "character": 4}}
}

test_preparerename_not_assigned_variable_not_supported if {
	policy := `package p

allow if input.user == "alice"`

	resp := preparerename.result.response
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 9}
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/prepareRename"

	resp == null
}

test_preparerename_rule_and_package_names[where] if {
	policy_a := `package rules

allow if input.admin`
	policy_b := `package app

import data.rules

x if rules.allow`

	policies := {"file:///a.rego": policy_a, "file:///b.rego": policy_b}
	modules := {file: regal.parse_module(file, src) | some file, src in policies}

	some where, case in {
		"rule declaration": {"uri": "file:///a.rego", "position": [2, 1], "range": [2, 0, 2, 5]},
		"rule use through import": {"uri": "file:///b.rego", "position": [4, 12], "range": [4, 11, 4, 16]},
		"package line": {"uri": "file:///a.rego", "position": [0, 10], "range": [0, 8, 0, 13]},
		"package import": {"uri": "file:///b.rego", "position": [2, 14], "range": [2, 12, 2, 17]},
	}

	resp := preparerename.result.response with data.workspace.parsed as modules
		with input.params.textDocument.uri as case.uri
		with input.params.position as {"line": case.position[0], "character": case.position[1]}
		with input.regal.file.lines as split(policies[case.uri], "\n")
		with input.method as "textDocument/prepareRename"

	resp == {
		"start": {"line": case.range[0], "character": case.range[1]},
		"end": {"line": case.range[2], "character": case.range[3]},
	}
}

test_preparerename_ignores_references_in_other_files if {
	policy_a := `package p

f(abcd) := 1`
	policy_b := `package p

f(x, abcd) := 2`

	resp := preparerename.result.response with data.workspace.parsed as {
		"file:///a.rego": regal.parse_module("a.rego", policy_a),
		"file:///b.rego": regal.parse_module("b.rego", policy_b),
	}
		with input.params.textDocument.uri as "file:///a.rego"
		with input.params.position as {"line": 2, "character": 5}
		with input.regal.file.lines as split(policy_a, "\n")
		with input.method as "textDocument/prepareRename"

	resp == {"start": {"line": 2, "character": 2}, "end": {"line": 2, "character": 6}}
}
