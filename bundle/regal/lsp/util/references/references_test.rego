package regal.lsp.util.references_test

import data.regal.lsp.testing
import data.regal.lsp.util.references

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

	refs := references.refs with data.workspace.parsed as {
		"file:///a.rego": regal.parse_module("a.rego", policy_a),
		"file:///b.rego": regal.parse_module("b.rego", policy_b),
	}
		with input.params.textDocument.uri as "file:///a.rego"
		with input.params.position as {"line": 2, "character": 6}
		with input.regal.file.lines as split(policy_a, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 3, "character": 30}, "end": {"line": 3, "character": 34}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 6}, "end": {"line": 2, "character": 10}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 11}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 26}, "end": {"line": 4, "character": 30}},
			"kind": references.kind_read,
		},
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

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 6}, "end": {"line": 3, "character": 10}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 5}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 10}, "end": {"line": 5, "character": 14}},
			"kind": references.kind_read,
		},
	}
}

test_references_no_match if {
	policy := `package p

foo := 42`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 11}
		with input.regal.file.lines as split(policy, "\n")

	refs == set()
}

test_references_every_var if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 7}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 8}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 2}, "end": {"line": 4, "character": 3}},
			"kind": references.kind_read,
		},
	}
}

test_references_comprehension_var if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
		x > 0
	]
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 12}, "end": {"line": 3, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 8}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 2}, "end": {"line": 5, "character": 3}},
			"kind": references.kind_read,
		},
	}
}

test_references_arg_without_declaration_by_default if {
	policy := `package p

greet(name) := msg if {
	msg := sprintf("hello, %s", [name])
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 6}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 30}, "end": {"line": 3, "character": 34}},
			"kind": references.kind_read,
		},
	}
}

test_references_some_var_without_declaration_when_false if {
	policy := `package p

list_admins contains user.name if {
	some user in input.users
	user.role == "admin"
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 6}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": false}

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 2, "character": 21}, "end": {"line": 2, "character": 25}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 5}},
			"kind": references.kind_read,
		},
	}
}

test_references_every_var_without_declaration_by_default if {
	policy := `package p

r if {
	every k, v in input.items {
		k > 0
		v.active
	}
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 2}, "end": {"line": 4, "character": 3}},
			"kind": references.kind_read,
		},
	}
}

test_references_comprehension_var_without_declaration_by_default if {
	policy := `package p

r := result if {
	result := [x |
		some x in input.items
		x > 0
	]
}`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 7}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 12}, "end": {"line": 3, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 2}, "end": {"line": 5, "character": 3}},
			"kind": references.kind_read,
		},
	}
}

test_references_arg_from_use_in_rule_head if {
	policy := `package p

fn(x) := x.y`

	refs := references.refs with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 9}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 2, "character": 9}, "end": {"line": 2, "character": 10}},
			"kind": references.kind_read,
		},
	}
}

test_references_assign_var_from_declaration if {
	policy := `package p

allow if {
	foo := 1
	bar := 2
	bar == foo
	foo != bar
}

other if {
	foo := 3
	foo == 3
}`

	refs := references.refs
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 3, "character": 1}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 8}, "end": {"line": 5, "character": 11}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 6, "character": 1}, "end": {"line": 6, "character": 4}},
			"kind": references.kind_read,
		},
	}
}

test_references_assign_var_from_use if {
	policy := `package p

allow if {
	foo := 1
	bar := 2
	bar == foo
	foo != bar
}`

	refs := references.refs
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 5, "character": 1}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 5, "character": 1}, "end": {"line": 5, "character": 4}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 6, "character": 8}, "end": {"line": 6, "character": 11}},
			"kind": references.kind_read,
		},
	}
}

test_references_assign_var_with_declaration if {
	policy := `package p

allow if {
	foo := 1
	foo == 1
}`

	refs := references.refs
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 4, "character": 1}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 3, "character": 1}, "end": {"line": 3, "character": 4}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 4}},
			"kind": references.kind_read,
		},
	}
}

test_references_assign_var_shadowing_import if {
	policy := `package p

import data.pkgb

allow if {
	pkgb := {"b": 2}
	pkgb.b == 2
}`

	refs := references.refs
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 5, "character": 1}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{
			"uri": "file:///p.rego",
			"range": {"start": {"line": 6, "character": 1}, "end": {"line": 6, "character": 5}},
			"kind": references.kind_read,
		},
	}
}
