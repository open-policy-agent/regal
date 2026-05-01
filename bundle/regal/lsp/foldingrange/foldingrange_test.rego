package regal.lsp.foldingrange_test

import data.regal.lsp.foldingrange

comment_blocks_module := regal.parse_module("p.rego", `

# METADATA
# title: package
# description: package-level metadata
package p

	# comment block
	# but not a
	# metadata block

# single-line comment`)

rules_module := regal.parse_module("p.rego", `package p

multiline_rule if {
	input.x
	input.y
	input.z
}

another if {
	input.a
	input.b
}

single_line_rule if input.c

single_line_rule_2 if { input.d }`)

test_folding_imports if {
	module := regal.parse_module("p.rego", `package p

import data.a
import data.b.c as c
import data.d.e.f

import input.g
import data.h
`)

	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.workspace.parsed["file:///p.rego"] as module

	result.response == {{
		"startLine": 2,
		"startCharacter": 0,
		"endLine": 7,
		"endCharacter": 12,
		"kind": "imports",
	}}
}

test_folding_imports_only_lines if {
	module := regal.parse_module("p.rego", `package p

import data.a
import data.b.c as c
import data.d.e.f

import input.g
import data.h
`)

	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.client.capabilities.textDocument.foldingRange.lineFoldingOnly as true
		with data.workspace.parsed["file:///p.rego"] as module

	result.response == {{
		"startLine": 2,
		"endLine": 7,
		"kind": "imports",
	}}
}

test_folding_comment_blocks if {
	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.workspace.parsed["file:///p.rego"] as comment_blocks_module

	result.response == {
		{
			"startLine": 2,
			"startCharacter": 0,
			"endLine": 4,
			"endCharacter": 37,
			"kind": "comment",
		},
		{
			"startLine": 7,
			"startCharacter": 1,
			"endLine": 9,
			"endCharacter": 17,
			"kind": "comment",
		},
	}
}

test_folding_comment_blocks_only_lines if {
	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.client.capabilities.textDocument.foldingRange.lineFoldingOnly as true
		with data.workspace.parsed["file:///p.rego"] as comment_blocks_module

	result.response == {
		{
			"startLine": 2,
			"endLine": 4,
			"kind": "comment",
		},
		{
			"startLine": 7,
			"endLine": 9,
			"kind": "comment",
		},
	}
}

test_folding_rule_blocks[name] if {
	tests := {
		"lines and characters": {
			"only_lines": false,
			"expected": {
				{
					"startLine": 2,
					"startCharacter": 0,
					"endLine": 5,
					"kind": "region",
				},
				{
					"startLine": 8,
					"startCharacter": 0,
					"endLine": 10,
					"kind": "region",
				},
			},
		},
		"lines only": {
			"only_lines": true,
			"expected": {
				{
					"startLine": 2,
					"endLine": 5,
					"kind": "region",
				},
				{
					"startLine": 8,
					"endLine": 10,
					"kind": "region",
				},
			},
		},
	}

	some name, test in tests

	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.client.capabilities.textDocument.foldingRange.lineFoldingOnly as test.only_lines
		with data.workspace.parsed["file:///p.rego"] as rules_module

	result.response == test.expected
}

test_ast_nodes_in_rules if {
	nodes := [
		`obj := {
			"a": 1,
			"b": 2,
		}`,
		`arr := [
			1,
			2,
		]`,
		`set := {
			1,
			2,
		}`,
		`arrcomp := [x |
			some x in input.arr
			x > 0
		]`,
		`objcomp := {k: v |
			some k, v in input.obj
			k != v
		}`,
		`setcomp := {x |
			some x in input.arr
			x > 0
		}`,
	]

	some node in nodes

	module := regal.parse_module("p.rego", $`package p

rule if \{
	{node}
}
`)

	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.workspace.parsed["file:///p.rego"] as module

	result.response == {
		# the rule
		{
			"startLine": 2,
			"startCharacter": 0,
			"endLine": 6,
			"kind": "region",
		},
		# the node
		{
			"startLine": 3,
			"startCharacter": max([indexof(node, "["), indexof(node, "{")]) + 1,
			"endLine": 5,
			"kind": "region",
		},
	}
}

test_folding_limit_honored[limit] if {
	some [limit, exp] in [
		[1, 1],
		[2, 2],
		[3, 2],
	]

	result := foldingrange.result
		with input.params.textDocument.uri as "file:///p.rego"
		with data.client.capabilities.textDocument.foldingRange.rangeLimit as limit
		with data.workspace.parsed["file:///p.rego"] as comment_blocks_module

	count(result.response) == exp
}
