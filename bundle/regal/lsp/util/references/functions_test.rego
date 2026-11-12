package regal.lsp.util.references_test

import data.regal.lsp.testing
import data.regal.lsp.util.references

test_function_call_bare_name_and_full_path if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

is_root|(u) if u == "root"

check if is_root(input.user)`,
		"file:///b.rego": `package q

call if data.p.is_root(input.user)`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 7}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 16}},
			"kind": references.kind_read,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 22}},
			"kind": references.kind_read,
		},
	}
}

test_function_from_import_alias_call if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

is_root(u) if u == "root"`,
		"file:///b.rego": `package q

import data.p as pkg

call := pkg.is_root|(input.user)`,
	})

	refs := references.refs
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{
			"uri": "file:///a.rego",
			"range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 7}},
			"kind": references.kind_write,
		},
		{
			"uri": "file:///b.rego",
			"range": {"start": {"line": 4, "character": 12}, "end": {"line": 4, "character": 19}},
			"kind": references.kind_read,
		},
	}
}
