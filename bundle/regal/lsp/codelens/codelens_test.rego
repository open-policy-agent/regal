package regal.lsp.codelens_test

import data.regal.lsp.codelens

test_code_lenses_for_module if {
	policy := `
	package foo

	import rego.v2

	rule1 := 1

	rule2 if 1 + rule1 == 2
	`

	module := regal.parse_module("policy.rego", policy)
	lenses := codelens.lenses
		with data.workspace.parsed as {"file://policy.rego": module}
		with input.regal as module.regal
		with input.params.textDocument.uri as "file://policy.rego"
		with data.server.feature_flags.debug_provider as true
		with data.client.init_options.enableDebugCodelens as true

	lenses == _expected_lenses
}

_expected_lenses := [
	{
		"command": {
			"arguments": [json.marshal({
				"target": "file://policy.rego",
				"path": "data.foo",
				"row": 2,
			})],
			"command": "regal.eval",
			"title": "Evaluate",
		},
		"range": {"end": {"character": 8, "line": 1}, "start": {"character": 1, "line": 1}},
	},
	{
		"command": {
			"arguments": [json.marshal({
				"target": "file://policy.rego",
				"path": "data.foo.rule1",
				"row": 6,
			})],
			"command": "regal.eval",
			"title": "Evaluate",
		},
		"range": {"end": {"character": 11, "line": 5}, "start": {"character": 1, "line": 5}},
	},
	{
		"command": {
			"arguments": [json.marshal({
				"target": "file://policy.rego",
				"path": "data.foo.rule2",
				"row": 8,
			})],
			"command": "regal.eval", "title": "Evaluate",
		},
		"range": {"end": {"character": 24, "line": 7}, "start": {"character": 1, "line": 7}},
	},
	{
		"command": {
			"arguments": [json.marshal({
				"target": "file://policy.rego",
				"path": "data.foo",
				"row": 2,
			})],
			"command": "regal.debug",
			"title": "Debug",
		},
		"range": {"end": {"character": 8, "line": 1}, "start": {"character": 1, "line": 1}},
	},
	{
		"command": {
			"arguments": [json.marshal({
				"target": "file://policy.rego",
				"path": "data.foo.rule2",
				"row": 8,
			})],
			"command": "regal.debug",
			"title": "Debug",
		},
		"range": {"end": {"character": 24, "line": 7}, "start": {"character": 1, "line": 7}},
	},
]
