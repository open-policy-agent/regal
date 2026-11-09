package regal.lsp.references_test

import data.regal.lsp.references
import data.regal.lsp.testing

test_function_call_bare_name_and_full_path if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

is_root|(u) if u == "root"

check if is_root(input.user)`,
		"file:///b.rego": `package q

call if data.p.is_root(input.user)`,
	})

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///a.rego", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 7}}},
		{"uri": "file:///a.rego", "range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 16}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 22}}},
	}
}

test_function_call_through_import_alias if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

is_root|(u) if u == "root"`,
		"file:///b.rego": `package q

import data.p as pkg

call if pkg.is_root(input.user)`,
	})

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///a.rego", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 7}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 4, "character": 12}, "end": {"line": 4, "character": 19}}},
	}
}
