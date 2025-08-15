package regal.lsp.codelens_test

import data.regal.lsp.codelens

# regal ignore:rule-length
test_code_lenses_for_module if {
	policy := `
	package foo

	import rego.v1

	rule1 := 1

	rule2 if 1 + rule1 == 2
	`
	module := regal.parse_module("policy.rego", policy)
	lenses := codelens.lenses with input as {
		"params": {"textDocument": {"uri": "file://policy.rego"}},
		# Ugh, why did we make enableDebugCodelens camel case 😭
		"regal": {
			"client": {"init_options": {"enableDebugCodelens": true}},
			"file": {
				"name": "policy.rego",
				"lines": split(policy, "\n"),
			},
		},
	}
		with data.workspace.parsed as {"file://policy.rego": module}

	lenses == [
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
}
