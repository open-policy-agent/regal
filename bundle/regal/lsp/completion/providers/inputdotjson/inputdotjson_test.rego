package regal.lsp.completion.providers.inputdotjson_test

import data.regal.lsp.completion.providers.inputdotjson as provider

test_matching_input_suggestions if {
	items := provider.items
		with input as input_obj
		with data.workspace as workspace_obj

	items == {
		{
			"detail": "object",
			"kind": 6,
			"label": "input.request",
			"documentation": {
				"kind": "markdown",
				"value": "(inferred from [`input.json`](foo/bar/input.json))",
			},
			"textEdit": {
				"newText": "input.request",
				"range": {
					"end": {"character": 13, "line": 5},
					"start": {"character": 6, "line": 5},
				},
			},
		},
		{
			"detail": "string",
			"kind": 6,
			"label": "input.request.method",
			"documentation": {
				"kind": "markdown",
				"value": "(inferred from [`input.json`](foo/bar/input.json))",
			},
			"textEdit": {
				"newText": "input.request.method",
				"range": {
					"end": {"character": 13, "line": 5},
					"start": {"character": 6, "line": 5},
				},
			},
		},
		{
			"detail": "string",
			"kind": 6,
			"label": "input.request.url",
			"documentation": {
				"kind": "markdown",
				"value": "(inferred from [`input.json`](foo/bar/input.json))",
			},
			"textEdit": {
				"newText": "input.request.url",
				"range": {
					"end": {"character": 13, "line": 5},
					"start": {"character": 6, "line": 5},
				},
			},
		},
	}
}

test_provides_input_suggestions_in_import_position if {
	items := provider.items
		with input as {
			"params": {
				"textDocument": {
					"uri": "file:///example.rego",
				},
				"position": {
					"line": 2,
					"character": 8,
				},
			},
			"regal": {
				"environment": {
					"input_path": "foo/bar/input.json",
				},
				"file": {
					"lines": [
						"package p", "", "import i",
					],
				},
			},
		}
		with data.workspace.inputs as {
			"foo/bar/input.json": {
				"request": {
					"method": "GET",
					"url": "https://example.com",
				},
			},
		}

	[item.label | some item in items] == ["input.request", "input.request.method", "input.request.url"]
}

test_not_matching_input_suggestions if {
	input_obj_new_loc := object.union(input_obj, {
		"params": {
			"textDocument": {"uri": "file:///example.rego"},
			"position": {"line": 0, "character": 0},
		},
	})
	items := provider.items
		with input as input_obj_new_loc
		with data.workspace as workspace_obj

	items == set()
}

input_obj := {
	"params": {
		"textDocument": {
			"uri": "file:///example.rego",
		},
		"position": {
			"line": 5,
			"character": 11,
		},
	},
	"regal": {
		"environment": {
			"input_path": "foo/bar/input.json",
		},
		"file": {
			"lines": [
				"package p",
				"",
				"import rego.v1",
				"",
				"allow if {",
				"    f(input.r",
				"}",
			],
		},
	},
}

workspace_obj := {
	"inputs": {
		"foo/bar/input.json": {
			"user": {
				"name": {
					"first": "John",
					"last": "Doe",
				},
				"email": "john@doe.com",
				"roles": [
					{"name": "admin"},
					{"name": "user"},
				],
			},
			"request": {
				"method": "GET",
				"url": "https://example.com",
			},
		},
	},
}
