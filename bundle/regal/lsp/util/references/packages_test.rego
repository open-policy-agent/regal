package regal.lsp.util.references_test

import data.regal.lsp.testing
import data.regal.lsp.util.references

test_package_imports_plain_and_alias if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p|

allow if input.a`,
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
			"range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 10}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 4, "character": 11}, "end": {"line": 4, "character": 14}},
			"kind": references.kind_read,
		},
	}
}

test_package_declared_over_multiple_files if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p|

allow if input.a`,
		"file:///b.rego": `package p

deny if input.b`,
		"file:///c.rego": `package q

import data.p`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 0, "character": 8}, "end": {"line": 0, "character": 9}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 0, "character": 8}, "end": {"line": 0, "character": 9}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
	}
}

test_package_sub_package_is_separate if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p.sub|

allow if input.a`,
		"file:///b.rego": `package q

import data.p
import data.p.sub as s
import data.p.sub.foobar

parent if p.allow

child if s.allow

other if foobar.allow`,
		"file:///c.rego": `package p.sub.foobar

allow if input.c`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 0, "character": 8}, "end": {"line": 0, "character": 13}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 3, "character": 7}, "end": {"line": 3, "character": 17}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 8, "character": 9}, "end": {"line": 8, "character": 10}},
			"kind": references.kind_read,
		},
	}
}

test_package_shadowed_import_not_included if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p|

allow if input.a`,
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

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
	}
}
