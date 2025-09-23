package regal.lsp.completion.providers.builtins_test

import data.regal.lsp.completion.providers.builtins

test_simple_builtin_completion if {
	items := builtins.items with data.workspace.builtins as _builtins with input as {"regal": {
		"context": {"location": {"row": 4, "col": 11}},
		"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]},
	}}

	items == {
		{
			"detail": "built-in function",
			"documentation": {
				"kind": "markdown",
				"value": "### [count](https://www.openpolicyagent.org/docs/policy-reference/#builtin-aggregates-count)\n\n```rego\nn := count(collection)\n```\n\nCount takes a collection or string and returns the number of elements (or characters) in it.\n\n#### Arguments\n\n- `collection` any: the set/array/object/string to be counted\n\nReturns `n` of type `number`: the count of elements, key/val pairs, or characters, respectively.\n",
			},
			"kind": 3,
			"label": "count",
			"textEdit": {
				"newText": "count",
				"range": {
					"end": {"character": 10, "line": 3},
					"start": {"character": 9, "line": 3},
				},
			},
		},
		{
			"detail": "built-in function",
			"documentation": {
				"kind": "markdown",
				"value": "### [crypto.hmac.md5](https://www.openpolicyagent.org/docs/policy-reference/#builtin-crypto.hmac.md5-crypto-hmac-md5)\n\n```rego\ny := crypto.hmac.md5(x, key)\n```\n\nReturns a string representing the MD5 HMAC of the input message using the input key.\n\n#### Arguments\n\n- `x` string: input string\n- `key` string: key to use\n\nReturns `y` of type `string`: MD5-HMAC of `x`\n",
			},
			"kind": 3,
			"label": "crypto.hmac.md5",
			"textEdit": {
				"newText": "crypto.hmac.md5",
				"range": {
					"end": {"character": 10, "line": 3},
					"start": {"character": 9, "line": 3},
				},
			},
		},
	}
}

test_simple_builtin_completion_single_match if {
	items := builtins.items with data.workspace.builtins as _builtins with input as {"regal": {
		"context": {"location": {"row": 4, "col": 12}},
		"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := co",
			"}",
		]},
	}}

	items == {{
		"detail": "built-in function",
		"documentation": {
			"kind": "markdown",
			"value": "### [count](https://www.openpolicyagent.org/docs/policy-reference/#builtin-aggregates-count)\n\n```rego\nn := count(collection)\n```\n\nCount takes a collection or string and returns the number of elements (or characters) in it.\n\n#### Arguments\n\n- `collection` any: the set/array/object/string to be counted\n\nReturns `n` of type `number`: the count of elements, key/val pairs, or characters, respectively.\n",
		},
		"kind": 3,
		"label": "count",
		"textEdit": {
			"newText": "count",
			"range": {
				"end": {"character": 11, "line": 3},
				"start": {"character": 9, "line": 3},
			},
		},
	}}
}

test_simple_builtin_completion_single_match_longer_ref if {
	items := builtins.items with data.workspace.builtins as _builtins with input as {"regal": {
		"context": {"location": {"row": 4, "col": 18}},
		"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := crypto.h",
			"}",
		]},
	}}

	items == {{
		"detail": "built-in function",
		"documentation": {
			"kind": "markdown",
			"value": "### [crypto.hmac.md5](https://www.openpolicyagent.org/docs/policy-reference/#builtin-crypto.hmac.md5-crypto-hmac-md5)\n\n```rego\ny := crypto.hmac.md5(x, key)\n```\n\nReturns a string representing the MD5 HMAC of the input message using the input key.\n\n#### Arguments\n\n- `x` string: input string\n- `key` string: key to use\n\nReturns `y` of type `string`: MD5-HMAC of `x`\n",
		},
		"kind": 3,
		"label": "crypto.hmac.md5",
		"textEdit": {
			"newText": "crypto.hmac.md5",
			"range": {
				"end": {"character": 17, "line": 3},
				"start": {"character": 9, "line": 3},
			},
		},
	}}
}

test_no_completion_of_deprecated_builtin if {
	builtins_deprecated := [object.union(_builtins[0], {"deprecated": true})]
	items := builtins.items with data.workspace.builtins as builtins_deprecated with input as {"regal": {
		"context": {"location": {"row": 4, "col": 11}},
		"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]},
	}}

	count(items) == 0
}

test_no_completion_of_infix_builtin if {
	builtins_deprecated := [object.union(_builtins[0], {"infix": "🔄"})]
	items := builtins.items with data.workspace.builtins as builtins_deprecated with input as {"regal": {
		"context": {"location": {"row": 4, "col": 11}},
		"file": {"lines": [
			"package p",
			"",
			"allow if {",
			"    b := c",
			"}",
		]},
	}}

	count(items) == 0
}

test_no_completion_in_default_rule if {
	items := builtins.items with data.workspace.builtins as _builtins with input as {"regal": {
		"context": {"location": {"row": 3, "col": 17}},
		"file": {"lines": [
			"package p",
			"",
			"default foo := c",
		]},
	}}

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
