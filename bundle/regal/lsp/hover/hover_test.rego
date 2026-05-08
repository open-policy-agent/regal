package regal.lsp.hover_test

import data.regal.lsp.hover

test_builtin_indexof if {
	res := hover.result.response
		with input as {
			"params": {"position": {
				"line": 0,
				"character": 4,
			}},
			"regal": {"file": {"lines": ["indexof([1,2,3], 2)"]}},
		}
		with data.workspace.builtins.indexof as {
			"name": "indexof",
			"categories": ["strings"],
			"decl": {
				"args": [
					{"name": "haystack", "type": "string", "description": "string to search in"},
					{"name": "needle", "type": "string", "description": "substring to look for"},
				],
				"result": {
					"name": "output",
					"type": "number",
					"description": "index of first occurrence, `-1` if not found",
				},
			},
			"description": "Returns the index of a substring contained inside a string.",
		}

	res.range.start.line == 0
	res.range.start.character == 0
	res.range.end.line == 0
	res.range.end.character == 7

	exp := concat("\n", [
		"### [indexof](https://www.openpolicyagent.org/docs/policy-reference/#builtin-strings-indexof)",
		"",
		"```rego",
		"output := indexof(haystack, needle)",
		"```",
		"",
		"Returns the index of a substring contained inside a string.",
		"",
		"#### Arguments",
		"",
		"- `haystack` string — string to search in",
		"- `needle` string — substring to look for",
		"",
		"#### Returns `output` of type `number`: index of first occurrence, `-1` if not found",
	])

	res.contents.value == exp
}

test_builtin_graph_reachable if {
	res := hover.result.response
		with input as {
			"params": {"position": {"line": 0, "character": 4}},
			"regal": {"file": {"lines": ["graph.reachable(graph, initial)"]}},
		}
		with data.workspace.builtins["graph.reachable"] as {
			"name": "graph.reachable",
			"categories": ["graph"],
			"decl": {
				"args": [
					{
						"name": "graph",
						"type": "object[any: any<array[any], set[any]>]",
						"description": "object containing a set or array of neighboring vertices",
					},
					{
						"name": "initial",
						"type": "any<array[any], set[any]>",
						"description": "set or array of root vertices",
					},
				],
				"result": {
					"name": "output",
					"type": "set[any]",
					"description": "set of vertices reachable from the `initial` vertices in the directed `graph`",
				},
			},
			"description": "Computes the set of reachable nodes in the graph from a set of starting nodes.",
		}

	res.range.start.line == 0
	res.range.start.character == 0
	res.range.end.line == 0
	res.range.end.character == 15

	exp := concat("\n", [
		"### [graph.reachable](https://www.openpolicyagent.org/docs/policy-reference/#builtin-graph-graphreachable)",
		"",
		"```rego",
		"output := graph.reachable(graph, initial)",
		"```",
		"",
		"Computes the set of reachable nodes in the graph from a set of starting nodes.",
		"",
		"#### Arguments",
		"",
		"- `graph` object[any: any<array[any], set[any]>] — object containing a set or array of neighboring vertices",
		"- `initial` any<array[any], set[any]> — set or array of root vertices",
		"",
		concat("", [
			"#### Returns `output` of type `set[any]`: ",
			"set of vertices reachable from the `initial` vertices in the directed `graph`",
		]),
	])

	res.contents.value == exp
}

test_builtin_json_filter if {
	dsc := concat("", [
		"Filters the object. For example: `json.filter({\"a\": {\"b\": \"x\", \"c\": \"y\"}}, [\"a/b\"])` ",
		"will result in `{\"a\": {\"b\": \"x\"}}`). ",
		"Paths are not filtered in-order and are deduplicated before being evaluated.",
	])
	res := hover.result.response
		with input as {
			"params": {"position": {"line": 2, "character": 11}},
			"regal": {"file": {"lines": ["foo", "bar", "allow if json.filter(object, paths)"]}},
		}
		with data.workspace.builtins["json.filter"] as {
			"name": "json.filter",
			"categories": ["object"],
			"decl": {
				"args": [
					{
						"name": "object",
						"type": "object[any: any]",
						"description": "object to filter",
					},
					{
						"name": "paths",
						"type": "any<array[any<string, array[any]>], set[any<string, array[any]>]>",
						"description": "JSON string paths",
					},
				],
				"result": {
					"name": "filtered",
					"type": "any",
					"description": "remaining data from `object` with only keys specified in `paths`",
				},
			},
			"description": dsc,
		}

	res.range.start.line == 2
	res.range.start.character == 9
	res.range.end.line == 2
	res.range.end.character == 20

	exp := concat("\n", [
		"### [json.filter](https://www.openpolicyagent.org/docs/policy-reference/#builtin-object-jsonfilter)",
		"",
		"```rego",
		"filtered := json.filter(object, paths)",
		"```",
		"",
		dsc,
		"",
		"#### Arguments",
		"",
		"- `object` object[any: any] — object to filter",
		"- `paths` any<array[any<string, array[any]>], set[any<string, array[any]>]> — JSON string paths",
		"",
		"#### Returns `filtered` of type `any`: remaining data from `object` with only keys specified in `paths`",
	])
	res.contents.value == exp
}

test_builtin_url_override if {
	res := hover.result.response
		with input as {
			"params": {"position": {
				"line": 0,
				"character": 0,
			}},
			"regal": {"file": {"lines": ["foo.bar(arg1, arg2)"]}},
		}
		with data.workspace.builtins["foo.bar"] as {
			"name": "foo.bar",
			"categories": ["foo", "url=https://example.com"],
			"decl": {
				"args": [
					{"name": "arg1", "type": "string", "description": "arg1 for foobar"},
					{"name": "arg2", "type": "string", "description": "arg2 for foobar"},
				],
				"result": {
					"name": "output",
					"type": "number",
					"description": "the output for foobar",
				},
			},
			"description": "Description for Foo Bar",
		}

	res.range.start.line == 0
	res.range.start.character == 0
	res.range.end.line == 0
	res.range.end.character == 7

	exp := concat("\n", [
		"### [foo.bar](https://example.com)",
		"",
		"```rego",
		"output := foo.bar(arg1, arg2)",
		"```",
		"",
		"Description for Foo Bar",
		"",
		"#### Arguments",
		"",
		"- `arg1` string — arg1 for foobar",
		"- `arg2` string — arg2 for foobar",
		"",
		"#### Returns `output` of type `number`: the output for foobar",
	])

	res.contents.value == exp
}

test_keyword_hover if {
	res := hover.result.response
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position.line as 2
		with input.params.position.character as 3
		with input.regal.file.lines as ["package p", "", "import data.foo"]
		with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", "package p\n\nimport data.foo")

	res.range.start.line == 2
	res.range.start.character == 0
	res.range.end.line == 2
	res.range.end.character == 5

	exp := "[View Usage Examples](https://www.openpolicyagent.org/docs/policy-reference//keywords/import)\n\n"

	res.contents.value == exp
}
