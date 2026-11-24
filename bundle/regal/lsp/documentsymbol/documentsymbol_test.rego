package regal.lsp.documentsymbol_test

import data.regal.lsp.documentsymbol

test_package_symbol if {
	module := regal.parse_module("bar.rego", "package foo.bar\n\n")

	r := documentsymbol.result.response
		with data.workspace.parsed["file:///foo/bar.rego"] as module
		with input.params.textDocument.uri as "file:///foo/bar.rego"
		with input.regal as module.regal

	r == [{
		"name": "data.foo.bar",
		"kind": 4,
		"range": {
			"start": {"character": 0, "line": 0},
			"end": {"character": 0, "line": 2},
		},
		"selectionRange": {
			"start": {"character": 0, "line": 0},
			"end": {"character": 0, "line": 2},
		},
	}]
}

test_rule_groups if {
	module := regal.parse_module("bar.rego", `package foo.bar
	rule0 if true

	rule1(x) if x + 1 == 2
	rule1(x) if {
		x < input.foo
		input.bar
	}

	rule2 if input.baz
	rule2 if input.qux
	rule2 if {
		rule1
		input.x
		input.y == 1
	}`)

	r := documentsymbol.result.response
		with data.workspace.parsed["file:///foo/bar.rego"] as module
		with input.params.textDocument.uri as "file:///foo/bar.rego"
		with input.regal as module.regal

	r == [_expected_rule_groups]
}

test_rule_detail_builtin_call_assignment if {
	module := regal.parse_module("bar.rego", `package foo.bar
	rule0 := graph.reachable(input.x, input.y)
	`)

	r := documentsymbol.result.response
		with data.workspace.parsed["file:///foo/bar.rego"] as module
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
		with input.params.textDocument.uri as "file:///foo/bar.rego"
		with input.regal as module.regal

	child := r[0].children[0]

	child.detail == "single-value rule (set)"
}

test_rule_kind_constant if {
	module := regal.parse_module("bar.rego", `package foo.bar
	pi := 3.14
	`)
	r := documentsymbol.result.response
		with data.workspace.parsed["file:///foo/bar.rego"] as module
		with input.params.textDocument.uri as "file:///foo/bar.rego"
		with input.regal as module.regal

	child := r[0].children[0]

	child.kind == 14 # constant
}

_expected_rule_groups := {
	"kind": 4,
	"name": "data.foo.bar",
	"range": {"end": {"character": 2, "line": 15}, "start": {"character": 0, "line": 0}},
	"selectionRange": {"end": {"character": 2, "line": 15}, "start": {"character": 0, "line": 0}},
	"children": [
		{
			"detail": "function(x) (boolean)", "kind": 12, "name": "rule1",
			"range": {"end": {"character": 2, "line": 7}, "start": {"character": 1, "line": 3}},
			"selectionRange": {"end": {"character": 2, "line": 7}, "start": {"character": 1, "line": 3}},
			"children": [
				{
					"detail": "function(x) (boolean)", "kind": 12, "name": "#1",
					"range": {"end": {"character": 23, "line": 3}, "start": {"character": 1, "line": 3}},
					"selectionRange": {"end": {"character": 23, "line": 3}, "start": {"character": 1, "line": 3}},
				},
				{
					"detail": "function(x) (boolean)", "kind": 12, "name": "#2",
					"range": {"end": {"character": 2, "line": 7}, "start": {"character": 1, "line": 4}},
					"selectionRange": {"end": {"character": 2, "line": 7}, "start": {"character": 1, "line": 4}},
				},
			],
		},
		{
			"detail": "single-value rule (boolean)", "kind": 13, "name": "rule2",
			"range": {"end": {"character": 2, "line": 15}, "start": {"character": 1, "line": 9}},
			"selectionRange": {"end": {"character": 2, "line": 15}, "start": {"character": 1, "line": 9}},
			"children": [
				{
					"detail": "single-value rule (boolean)", "kind": 13, "name": "#1",
					"range": {"end": {"character": 19, "line": 9}, "start": {"character": 1, "line": 9}},
					"selectionRange": {"end": {"character": 19, "line": 9}, "start": {"character": 1, "line": 9}},
				},
				{
					"detail": "single-value rule (boolean)", "kind": 13, "name": "#2",
					"range": {"end": {"character": 19, "line": 10}, "start": {"character": 1, "line": 10}},
					"selectionRange": {"end": {"character": 19, "line": 10}, "start": {"character": 1, "line": 10}},
				},
				{
					"detail": "single-value rule (boolean)", "kind": 13, "name": "#3",
					"range": {"end": {"character": 2, "line": 15}, "start": {"character": 1, "line": 11}},
					"selectionRange": {"end": {"character": 2, "line": 15}, "start": {"character": 1, "line": 11}},
				},
			],
		},
		{
			"detail": "single-value rule (boolean)", "kind": 13, "name": "rule0",
			"range": {"end": {"character": 14, "line": 1}, "start": {"character": 1, "line": 1}},
			"selectionRange": {"end": {"character": 14, "line": 1}, "start": {"character": 1, "line": 1}},
		},
	],
}
