package regal.lsp.completion.providers.builtins_test

import data.regal.lsp.completion.providers.builtins

test_simple_builtin_completion if {
	items := builtins.items with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 10},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]}},
	}
		with data.workspace.builtins as _builtins

	items == {
		{
			"detail": "built-in function",
			"kind": 3,
			"label": "count",
			"textEdit": {
				"newText": "count",
				"range": {
					"end": {"character": 10, "line": 3},
					"start": {"character": 9, "line": 3},
				},
			},
			"data": {"resolver": "builtins"},
		},
		{
			"detail": "built-in function",
			"kind": 3,
			"label": "crypto.hmac.md5",
			"textEdit": {
				"newText": "crypto.hmac.md5",
				"range": {
					"end": {"character": 10, "line": 3},
					"start": {"character": 9, "line": 3},
				},
			},
			"data": {"resolver": "builtins"},
		},
	}
}

test_simple_builtin_completion_single_match if {
	items := builtins.items with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 11},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := co",
			"}",
		]}},
	}
		with data.workspace.builtins as _builtins

	items == {{
		"detail": "built-in function",
		"kind": 3,
		"label": "count",
		"textEdit": {
			"newText": "count",
			"range": {
				"end": {"character": 11, "line": 3},
				"start": {"character": 9, "line": 3},
			},
		},
		"data": {"resolver": "builtins"},
	}}
}

test_simple_builtin_completion_single_match_longer_ref if {
	items := builtins.items with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 17},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := crypto.h",
			"}",
		]}},
	}
		with data.workspace.builtins as _builtins

	items == {{
		"detail": "built-in function",
		"kind": 3,
		"label": "crypto.hmac.md5",
		"textEdit": {
			"newText": "crypto.hmac.md5",
			"range": {
				"end": {"character": 17, "line": 3},
				"start": {"character": 9, "line": 3},
			},
		},
		"data": {"resolver": "builtins"},
	}}
}

test_no_completion_of_deprecated_builtin if {
	builtins_deprecated := [object.union(_builtins[0], {"deprecated": true})]
	items := builtins.items with data.workspace.builtins as builtins_deprecated with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 10},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]}},
	}

	count(items) == 0
}

test_no_completion_of_infix_builtin if {
	builtins_deprecated := [object.union(_builtins[0], {"infix": "🔄"})]
	items := builtins.items with data.workspace.builtins as builtins_deprecated with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 10},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]}},
	}

	count(items) == 0
}

test_no_completion_in_default_rule if {
	items := builtins.items with input as {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 2, "character": 16},
		},
		"regal": {"file": {"lines": [
			"package p",
			"",
			"default foo := c",
		]}},
	}
		with data.workspace.builtins as _builtins

	count(items) == 0
}

_builtins := [
	{
		"name": "count",
		"description": "Count takes a collection or string and returns the number of elements (or characters) in it.",
		"categories": ["aggregates"],
		"decl": {
			"args": [{
				"description": "the set/array/object/string to be counted",
				"name": "collection",
				"of": [
					{"type": "string"},
					{
						"dynamic": {"type": "any"},
						"type": "array",
					},
					{
						"dynamic": {
							"key": {"type": "any"},
							"value": {"type": "any"},
						},
						"type": "object",
					},
					{
						"of": {"type": "any"},
						"type": "set",
					},
				],
				"type": "any",
			}],
			"result": {
				"description": "the count of elements, key/val pairs, or characters, respectively.",
				"name": "n",
				"type": "number",
			},
			"type": "function",
		},
	},
	{
		"name": "crypto.hmac.md5",
		"description": "Returns a string representing the MD5 HMAC of the input message using the input key.",
		"decl": {
			"args": [
				{
					"description": "input string",
					"name": "x",
					"type": "string",
				},
				{
					"description": "key to use",
					"name": "key",
					"type": "string",
				},
			],
			"result": {
				"description": "MD5-HMAC of `x`",
				"name": "y",
				"type": "string",
			},
			"type": "function",
		},
	},
]
