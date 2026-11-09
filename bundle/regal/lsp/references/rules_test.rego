package regal.lsp.references_test

import data.regal.lsp.references
import data.regal.lsp.testing

# the other heads of a rule are part of its declaration, so they are returned as references
test_rule_declarations_across_files if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

default allow := false

allow| if input.a`,
		"file:///b.rego": `package p

allow if input.b`,
	})

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///a.rego", "range": {"start": {"line": 2, "character": 8}, "end": {"line": 2, "character": 13}}},
		{"uri": "file:///a.rego", "range": {"start": {"line": 4, "character": 0}, "end": {"line": 4, "character": 5}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}}},
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

	refs := references.result.response
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 20}}},
	}
}

test_rule_from_bare_name_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package p

other if allow|`,
	})

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///a.rego", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}}},
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 9}, "end": {"line": 2, "character": 14}}},
	}
}

test_rule_from_full_path_use if {
	[tdp, modules] := testing.at_cursor_in({
		"file:///a.rego": `package p

allow if input.a`,
		"file:///b.rego": `package q

full if data.p.allow|`,
	})

	refs := references.result.response
		with input as tdp
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 15}, "end": {"line": 2, "character": 20}}},
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

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed as modules

	refs == {
		{"uri": "file:///b.rego", "range": {"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}}},
		{"uri": "file:///c.rego", "range": {"start": {"line": 2, "character": 21}, "end": {"line": 2, "character": 26}}},
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

	refs := references.result.response
		with input as tdp
		with data.workspace.parsed as modules

	refs == set()
}
