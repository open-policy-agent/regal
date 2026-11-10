package regal.lsp.util.references_test

import data.regal.lsp.testing
import data.regal.lsp.util.references

# the other heads of a rule are part of its declaration, so they are returned as references
test_rule_declarations_across_files if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

default allow := false

allow| if input.a`,
		"file:///b.rego": `package p

allow if input.b`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 8}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 4, "character": 0}, "end": {"line": 4, "character": 5}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
			"kind": references.kind_write,
		},
	}
}

test_rule_full_path_use_cross_file if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow| if input.a`,
		"file:///b.rego": `package q

full if data.p.allow

nested if data.p.sub.allow`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 20}},
			"kind": references.kind_read,
		},
	}
}

test_rule_from_bare_name_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package p

other if allow|`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 9}, "end": {"line": 2, "character": 14}},
			"kind": references.kind_read,
		},
	}
}

test_rule_from_full_path_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package q

full if data.p.allow|`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 20}},
			"kind": references.kind_read,
		},
	}
}

test_rule_parent_package_excluded if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a

other if allow`,
		"file:///b.rego": `package p.sub

allow| := true`,
		"file:///c.rego": `package q

nested if data.p.sub.allow`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 2, "character": 21}, "end": {"line": 2, "character": 26}},
			"kind": references.kind_read,
		},
	}
}

test_rule_shadowed_by_local_name_not_included if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow| if input.a`,
		"file:///b.rego": `package p

shadowed if {
	allow := 1
	allow == 1
}

fn(allow) := allow`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == set()
}

test_rule_from_import_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package q

import data.p

plain if p.allow|`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 11}, "end": {"line": 4, "character": 16}},
			"kind": references.kind_read,
		},
	}
}

test_rule_use_through_import_and_alias_cross_file if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow| if input.a`,
		"file:///b.rego": `package q

import data.p

plain if p.allow`,
		"file:///c.rego": `package r

import data.p as pkg

aliased if pkg.allow`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 11}, "end": {"line": 4, "character": 16}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 4, "character": 15}, "end": {"line": 4, "character": 20}},
			"kind": references.kind_read,
		},
	}
}

test_rule_import_use_shadowed_by_local_name_not_included if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow| if input.a`,
		"file:///b.rego": `package q

import data.p

shadowed(p) := p.allow

assigned if {
	p := input.x
	p.allow
}`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == set()
}

test_rule_import_use_of_other_package_or_rule_not_included if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow| if input.a`,
		"file:///b.rego": `package q

allow if input.b`,
		"file:///c.rego": `package r

import data.p
import data.q

other if q.allow

missing if p.nope`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == set()
}

test_rule_from_import_use_shadowed_by_local_name_not_found if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package q

import data.p

shadowed(p) := p.allow|`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == set()
}

test_rule_from_import_use_unknown_name_not_found if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package q

import data.p

missing if p.nope|`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == set()
}
