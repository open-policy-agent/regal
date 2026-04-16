package regal.lsp.semantictokens_test

import data.regal.lsp.semantictokens

test_result if {
	policy := `package regal.woo

import data.regal.ast

test_function(param1, param2) := result if {
	ast.is_constant
}`
	module := regal.parse_module("p.rego", policy)
	result := semantictokens.result with data.workspace.parsed["file:///p.rego"] as module
		with input.regal as module.regal
		with input.params.textDocument.uri as "file:///p.rego"

	result == {"response": {
		"imports": {{"location": "3:19:3:22", "type": "string", "value": "ast"}},
		"packages": {
			{"col": 0, "length": 7, "line": 0, "modifiers": 0, "type": 3},
			{"col": 14, "length": 3, "line": 0, "modifiers": 0, "type": 0},
		},
		"vars": {
			"comprehensions": {
				"declaration": set(),
				"reference": set(),
			},
			"every_expr": {
				"declaration": set(),
				"reference": set(),
			},
			"function_args": {
				"declaration": {
					{"location": "5:15:5:21", "type": "var", "value": "param1"},
					{"location": "5:23:5:29", "type": "var", "value": "param2"},
				},
				"reference": set(),
			},
			"some_expr": {"declaration": set(), "reference": set()},
		},
	}}
}
