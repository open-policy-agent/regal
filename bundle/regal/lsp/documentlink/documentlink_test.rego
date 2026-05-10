package regal.lsp.documentlink_test

import data.regal.lsp.documentlink

test_documentlink_ranges_in_inline_ignores if {
	lines := [
		"package p",
		"",
		"# regal ignore:messy-rule,unresolved-reference",
		"ignored if directives",
	]

	items := documentlink.items
		with input.params.textDocument.uri as "file://p.rego"
		with input.regal.file.lines as lines
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", concat("\n", lines))
		with data.workspace.config.rules as {
			"style": {"messy-rule": {}},
			"imports": {"unresolved-reference": {}},
		}

	items == {
		{
			"range": {
				"end": {
					"character": 25,
					"line": 2,
				},
				"start": {
					"character": 15,
					"line": 2,
				},
			},
			"target": "https://www.openpolicyagent.org/projects/regal/rules/style/messy-rule",
			"tooltip": "See documentation for messy-rule",
		},
		{
			"range": {
				"end": {
					"character": 46,
					"line": 2,
				},
				"start": {
					"character": 26,
					"line": 2,
				},
			},
			"target": "https://www.openpolicyagent.org/projects/regal/rules/imports/unresolved-reference",
			"tooltip": "See documentation for unresolved-reference",
		},
	}
}
