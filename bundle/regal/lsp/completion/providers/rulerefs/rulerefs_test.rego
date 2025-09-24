package regal.lsp.completion.providers.rulerefs_test

import data.regal.ast

import data.regal.lsp.completion.providers.rulerefs as provider

workspace := {
	"file:///current_file.rego": `package foo

import rego.v1

import data.imported_pkg
import data.imported_pkg_2

local_rule if true
`,
	"imported_file.rego": `package imported_pkg

another_rule := true

_internal_rule := true
`,
	"imported_file_2.rego": `package imported_pkg_2

another_rule_2 := true

_internal_rule_2 := true
`,
	"not_imported_pkg.rego": `package not_imported_pkg.foo.bar

yet_another_rule := false

_internal_rule := true
`,
}

parsed_modules[file_uri] := regal.parse_module(file_uri, contents) if some file_uri, contents in workspace

defined_refs[file_uri] contains concat(".", [package_name, ast.ref_to_string(rule.head.ref)]) if {
	some file_uri, parsed_module in parsed_modules

	package_name := ast.ref_to_string(parsed_module.package.path)

	some rule in parsed_module.rules
}

test_rule_refs_no_word if {
	current_file_contents := concat("", [workspace["file:///current_file.rego"], `
another_local_rule := `])

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///current_file.rego"},
			"position": {"line": 9, "character": 20},
		},
		"regal": {"file": {"lines": split(current_file_contents, "\n")}},
	}

	items := provider.items with input as regal_module
		with data.workspace.parsed as parsed_modules
		with data.workspace.defined_refs as defined_refs

	labels := {item.label | some item in items}

	expected_refs := {
		"local_rule",
		"imported_pkg.another_rule",
		"imported_pkg_2.another_rule_2",
		"data.not_imported_pkg.foo.bar", # partial generated from rule below
		"data.not_imported_pkg.foo.bar.yet_another_rule",
	}

	expected_refs == labels
}

test_rule_refs_partial_word if {
	current_file_contents := concat("", [workspace["file:///current_file.rego"], `
another_local_rule := imp`])

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///current_file.rego"},
			"position": {"line": 9, "character": 25},
		},
		"regal": {"file": {"lines": split(current_file_contents, "\n")}},
	}

	items := provider.items with input as regal_module
		with data.workspace.parsed as parsed_modules
		with data.workspace.defined_refs as defined_refs

	labels := [item.label | some item in items]

	expected_refs := [
		"imported_pkg.another_rule",
		"imported_pkg_2.another_rule_2",
	]

	expected_refs == labels
}

test_rule_refs_not_in_rule if {
	current_file_contents := concat("", [workspace["file:///current_file.rego"], `

a`])

	lines := split(current_file_contents, "\n")

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///current_file.rego"},
			"position": {"line": count(lines) - 1, "character": 0},
		},
		"regal": {"file": {"lines": lines}},
	}

	items := provider.items with input as regal_module
		with data.workspace.parsed as parsed_modules
		with data.workspace.defined_refs as defined_refs

	count(items) == 0
}

test_rule_refs_no_recursion if {
	current_file_contents := concat("", [workspace["file:///current_file.rego"], `

local_rule if local`])

	lines := split(current_file_contents, "\n")

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///current_file.rego"},
			"position": {"line": count(lines) - 1, "character": 18},
		},
		"regal": {"file": {"lines": lines}},
	}

	items := provider.items with input as regal_module
		with data.workspace.parsed as parsed_modules
		with data.workspace.defined_refs as defined_refs

	count(items) == 0
}

test_rule_refs_no_recursion_func if {
	current_file_contents := concat("", [workspace["file:///current_file.rego"], `

local_fun("") := foo {
	true
}

local_func("foo") := local_f`])

	lines := split(current_file_contents, "\n")

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///current_file.rego"},
			"position": {"line": count(lines) - 1, "character": 25},
		},
		"regal": {"file": {"lines": lines}},
	}

	items := provider.items with input as regal_module
		with data.workspace.parsed as parsed_modules
		with data.workspace.defined_refs as defined_refs

	count(items) == 0
}
