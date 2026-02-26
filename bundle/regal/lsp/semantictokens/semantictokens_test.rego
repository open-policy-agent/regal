package regal.lsp.semantictokens_test

import data.regal.lsp.semantictokens

test_result if {
	policy := `package regal.woo

import data.regal.ast

test_function(param1, param2) := result if {
	ast.is_constant
}`
	result := semantictokens.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	result == {"response": {
		"imports": {{"location": "3:19:3:22", "type": "string", "value": "ast"}},
		"packages": {{"location": "1:15:1:18", "type": "string", "value": "woo"}},
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
