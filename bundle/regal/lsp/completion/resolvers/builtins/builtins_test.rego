package regal.lsp.completion.resolvers.builtins_test

test_resolve_builtin_documentation if {
	input_count := {"params": {"label": "count", "data": {"resolver": "builtins"}}}
	result_count := data.regal.lsp.completion.resolvers.builtins.resolve with input as input_count
		with data.workspace.builtins as _builtins

	result_count == {
		"label": "count",
		"documentation": {
			"kind": "markdown",
			"value": "### [count](https://www.openpolicyagent.org/docs/policy-reference/#builtin-aggregates-count)\n\n```rego\nn := count(collection)\n```\n\nCount takes a collection or string and returns the number of elements (or characters) in it.\n\n#### Arguments\n\n- `collection` any: the set/array/object/string to be counted\n\nReturns `n` of type `number`: the count of elements, key/val pairs, or characters, respectively.\n",
		},
		"data": {"resolver": "builtins"},
	}
	input_hmac := {"params": {"label": "crypto.hmac.md5", "data": {"resolver": "builtins"}}}
	result_hmac := data.regal.lsp.completion.resolvers.builtins.resolve with input as input_hmac
		with data.workspace.builtins as _builtins

	result_hmac == {
		"data": {"resolver": "builtins"},
		"documentation": {
			"kind": "markdown",
			"value": "### [crypto.hmac.md5](https://www.openpolicyagent.org/docs/policy-reference/#builtin-crypto.hmac.md5-crypto-hmac-md5)\n\n```rego\ny := crypto.hmac.md5(x, key)\n```\n\nReturns a string representing the MD5 HMAC of the input message using the input key.\n\n#### Arguments\n\n- `x` string: input string\n- `key` string: key to use\n\nReturns `y` of type `string`: MD5-HMAC of `x`\n",
		},
		"label": "crypto.hmac.md5",
	}
}

_builtins := {
	"count": {
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
	"crypto.hmac.md5": {
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
}
