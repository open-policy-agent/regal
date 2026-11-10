package regal.lsp.util.references_test

import data.regal.lsp.testing
import data.regal.lsp.util.references

test_package_imports_aliases_and_full_path_uses if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p|

allow if input.a`,
		"file:///b.rego": `package q

import data.p

plain if p.allow`,
		"file:///c.rego": `package r

import data.p as pkg

aliased if pkg.allow`,
		"file:///d.rego": `package s

full_path if data.p.allow

sub_package if data.p.sub.allow`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 12}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 10}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 2, "character": 12}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///c.rego",
			"range": {"start": {"line": 4, "character": 11}, "end": {"line": 4, "character": 14}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///d.rego",
			"range": {"start": {"line": 2, "character": 18}, "end": {"line": 2, "character": 19}},
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
			"range": {"start": {"line": 2, "character": 12}, "end": {"line": 2, "character": 13}},
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
			"range": {"start": {"line": 0, "character": 10}, "end": {"line": 0, "character": 13}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 3, "character": 14}, "end": {"line": 3, "character": 17}},
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
			"range": {"start": {"line": 2, "character": 12}, "end": {"line": 2, "character": 13}},
			"kind": references.kind_read,
		},
	}
}

test_package_document_highlight_skips_other_files if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p|

allow if input.a`,
		"file:///b.rego": `package p

deny if input.b`,
		"file:///c.rego": `package q

import data.p

via_import if p.allow`,
	})

	refs := references.refs
		with input as object.union(tdp, {"method": "textDocument/documentHighlight"})
		with data.workspace.parsed as modules

	refs == {{
		"uri": "file:///a.rego",
		"range": {"start": {"line": 0, "character": 8}, "end": {"line": 0, "character": 9}},
		"kind": references.kind_write,
	}}
}

test_package_rule_with_the_name_of_import_is_not_a_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package foo.bar|

allow if input.a`,
		"file:///b.rego": `package q

import data.foo.bar

bar := 1

use := bar`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 16}, "end": {"line": 2, "character": 19}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 6, "character": 7}, "end": {"line": 6, "character": 10}},
			"kind": references.kind_read,
		},
	}
}

test_package_uses_by_exact_path_deeper_imports_and_parent_imports if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package foo.bar|

allow if input.a`,
		"file:///b.rego": `package b

import data.foo
import data.foo as f

x := foo.bar.allow

y := f.bar.allow

z := foo.bar`,
		"file:///c.rego": `package c

import data.foo.bar.allow

x := data.foo.bar

y := data.foo.bar[input.k]`,
	})

	refs := references.refs
		with input as tdp
		with data.workspace.parsed as modules

	refs == {{
		"uri": file,
		"range": {"start": {"line": row, "character": col}, "end": {"line": row, "character": col + 3}},
		"kind": references.kind_read,
	} |
		some [file, row, col] in [
			["file:///b.rego", 5, 9],
			["file:///b.rego", 7, 7],
			["file:///b.rego", 9, 9],
			["file:///c.rego", 2, 16],
			["file:///c.rego", 4, 14],
			["file:///c.rego", 6, 14],
		]
	}
}
