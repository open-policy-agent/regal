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
		with input.method as "textDocument/rename"

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
		with input.method as "textDocument/rename"

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
		with input.method as "textDocument/rename"

	resp == null
}

test_rename_every_var if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 7}
		with input.params.newName as "key"
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/rename"

	resp == {"changes": {"file:///p.rego": {
		{"newText": "key", "range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 8}}},
		{"newText": "key", "range": {"start": {"line": 4, "character": 2}, "end": {"line": 4, "character": 3}}},
	}}}
}

test_rename_comprehension_var if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
		x > 0
	]
}`

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.params.newName as "item"
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/rename"

	resp == {"changes": {"file:///p.rego": {
		{"newText": "item", "range": {"start": {"line": 3, "character": 12}, "end": {"line": 3, "character": 13}}},
		{"newText": "item", "range": {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 8}}},
		{"newText": "item", "range": {"start": {"line": 5, "character": 2}, "end": {"line": 5, "character": 3}}},
	}}}
}

test_rename_rule_cross_file if {
	policy_a := `package rules

allow if input.admin

allow if is_root

is_root if input.user == "root"`
	policy_b := `package app

import data.rules
import data.rules as r

x if rules.allow

y if r.allow

z if data.rules.allow`
	policy_c := `package other

allow if true`
	policy_d := `package imports

import data.rules.allow
import data.rules.allow as a

r if allow

s if a`

	resp := rename.result.response with data.workspace.parsed as {
		"file:///a.rego": regal.parse_module("a.rego", policy_a),
		"file:///b.rego": regal.parse_module("b.rego", policy_b),
		"file:///c.rego": regal.parse_module("c.rego", policy_c),
		"file:///d.rego": regal.parse_module("d.rego", policy_d),
	}
		with input.params.textDocument.uri as "file:///a.rego"
		with input.params.position as {"line": 2, "character": 1}
		with input.params.newName as "permit"
		with input.regal.file.lines as split(policy_a, "\n")
		with input.method as "textDocument/rename"

	resp == {"changes": {
		"file:///a.rego": {
			{"newText": "permit", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}}},
			{"newText": "permit", "range": {"start": {"line": 4, "character": 0}, "end": {"line": 4, "character": 5}}},
		},
		"file:///b.rego": {
			{"newText": "permit", "range": {"start": {"line": 5, "character": 11}, "end": {"line": 5, "character": 16}}},
			{"newText": "permit", "range": {"start": {"line": 7, "character": 7}, "end": {"line": 7, "character": 12}}},
			{"newText": "permit", "range": {"start": {"line": 9, "character": 16}, "end": {"line": 9, "character": 21}}},
		},
		# a direct import is renamed, but the alias of one, and its uses, are not
		"file:///d.rego": {
			{"newText": "permit", "range": {"start": {"line": 2, "character": 18}, "end": {"line": 2, "character": 23}}},
			{"newText": "permit", "range": {"start": {"line": 3, "character": 18}, "end": {"line": 3, "character": 23}}},
			{"newText": "permit", "range": {"start": {"line": 5, "character": 5}, "end": {"line": 5, "character": 10}}},
		},
	}}
}

test_rename_new_name[new_name] if {
	policy := `package rules

allow if input.admin`

	some new_name, valid in {
		"x1": true,
		"allow_all": true,
		"_private": true,
		"Allow": true,
		"": false,
		"a b": false,
		"a.b": false,
		"1a": false,
		"_": false,
		"if": false,
		"package": false,
		"input": false,
	}

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 1}
		with input.params.newName as new_name
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/rename"

	resp == _renamed(new_name, valid)
}

_renamed(new_name, true) := {"changes": {"file:///p.rego": {{
	"newText": new_name,
	"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
}}}}

_renamed(_, false) := null

test_rename_package[from] if {
	policy_a := `package foo.bar

allow if input.a`
	policy_b := `package app

import data.foo.bar
import data.foo.bar as baz

x if bar.allow

y if baz.allow

z if data.foo.bar.allow`

	policies := {"file:///a.rego": policy_a, "file:///b.rego": policy_b}
	modules := {file: regal.parse_module(file, src) | some file, src in policies}

	some from, position in {
		"package line": ["file:///a.rego", 0, 13],
		"import": ["file:///b.rego", 2, 17],
		"aliased import": ["file:///b.rego", 3, 17],
	}

	resp := rename.result.response with data.workspace.parsed as modules
		with input.params.textDocument.uri as position[0]
		with input.params.position as {"line": position[1], "character": position[2]}
		with input.params.newName as "qux"
		with input.regal.file.lines as split(policies[position[0]], "\n")
		with input.method as "textDocument/rename"

	# the alias, and the uses of it, are left as they are
	resp == {"changes": {
		"file:///a.rego": {
			{"newText": "qux", "range": {"start": {"line": 0, "character": 12}, "end": {"line": 0, "character": 15}}},
		},
		"file:///b.rego": {
			{"newText": "qux", "range": {"start": {"line": 2, "character": 16}, "end": {"line": 2, "character": 19}}},
			{"newText": "qux", "range": {"start": {"line": 3, "character": 16}, "end": {"line": 3, "character": 19}}},
			{"newText": "qux", "range": {"start": {"line": 5, "character": 5}, "end": {"line": 5, "character": 8}}},
			{"newText": "qux", "range": {"start": {"line": 9, "character": 14}, "end": {"line": 9, "character": 17}}},
		},
	}}
}

test_rename_import_not_supported[name] if {
	policy := `package p

import data.ext

x := ext.y`

	some name, position in {"path": [2, 14], "use": [4, 7]}

	resp := rename.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": position[0], "character": position[1]}
		with input.params.newName as "zap"
		with input.regal.file.lines as split(policy, "\n")
		with input.method as "textDocument/rename"

	resp == null
}
