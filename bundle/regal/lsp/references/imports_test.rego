package regal.lsp.references_test

import data.regal.lsp.references
import data.regal.lsp.testing

test_references_import_uses_in_file if {
	policy := `package p

import data.pkgb
import data.pkgb.sub as s

wow := pkgb.b

other := s.c

bare := count(pkgb)

local if {
	pkgb := {"b": 2}
	pkgb.b == 2
}

fn(pkgb) := pkgb.b`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 10}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 5, "character": 7}, "end": {"line": 5, "character": 11}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 9, "character": 14}, "end": {"line": 9, "character": 18}}},
	}
}

test_references_import_with_declaration if {
	policy := `package p

import data.pkgb

wow := pkgb.b`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 10}
		with input.regal.file.lines as split(policy, "\n")
		with input.params.context as {"includeDeclaration": true}

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 16}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 11}}},
	}
}

test_references_import_with_alias if {
	policy := `package p

import data.pkgb.sub as s

other := s.c

wow := pkgb.b`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 10}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 10}}},
	}
}

test_references_import_used_as_ref_index_is_not_shadowing if {
	policy := `package p

import data.pkgb

idx := data.q[pkgb]

wow := pkgb.b`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 10}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 14}, "end": {"line": 4, "character": 18}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 6, "character": 7}, "end": {"line": 6, "character": 11}}},
	}
}

test_references_import_from_alias_position if {
	policy := `package p

import data.pkgb as wow

allow if wow.b == 1

allow if wow.b == 2`

	refs := references.result.response with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position as {"line": 2, "character": 21}
		with input.regal.file.lines as split(policy, "\n")

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 9}, "end": {"line": 4, "character": 12}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 6, "character": 9}, "end": {"line": 6, "character": 12}}},
	}
}

test_references_import_from_use if {
	snippet := `import data.pkgb

wow := pkgb|.b

local if {
	pkgb := 1
	pkgb == 1
}`

	[tdp, module] := testing.at_cursor(snippet)

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed["file:///p.rego"] as module

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 2, "character": 7}, "end": {"line": 2, "character": 16}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 4, "character": 7}, "end": {"line": 4, "character": 11}}},
	}
}

test_references_import_shadowed_by_local_variable_not_included if {
	snippet := `import data.pkgb

local if {
	pkgb := 1
	pkgb| == 1
}`

	[tdp, module] := testing.at_cursor(snippet)

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed["file:///p.rego"] as module

	refs == {
		{"uri": "file:///p.rego", "range": {"start": {"line": 5, "character": 1}, "end": {"line": 5, "character": 5}}},
		{"uri": "file:///p.rego", "range": {"start": {"line": 6, "character": 1}, "end": {"line": 6, "character": 5}}},
	}
}

test_references_import_unrelated_variable_not_included if {
	snippet := `import data.pkgb

allow if input|.user == "admin"`

	[tdp, module] := testing.at_cursor(snippet)

	refs := references.result.response
		with input as object.union(tdp, {"params": {"context": {"includeDeclaration": true}}})
		with data.workspace.parsed["file:///p.rego"] as module

	refs == set()
}
