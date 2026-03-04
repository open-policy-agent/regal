package regal.lsp.semantictokens.vars.some_expr_test

import data.regal.lsp.semantictokens.vars.some_expr

test_some_vars if {
	policy := `package regal.woo

some_two_vars if {
	some i, item in input.array   
	i < 10                        
	item > 0                        
}`

	some note, tc in {"some expression variables": {
		"declarations": {
			{"location": "4:10:4:14", "type": "var", "value": "item"},
			{"location": "4:7:4:8", "type": "var", "value": "i"},
		},
		"references": {
			{"location": "5:2:5:3", "type": "var", "value": "i"},
			{"location": "6:2:6:6", "type": "var", "value": "item"},
		},
	}}

	tokens := some_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	tc.declarations == tokens.declaration
	tc.references == tokens.reference
}

test_some_single_var_case if {
	policy := `package regal.woo

some_one_var if {
	some i in input.array   
	i < 10                                              
}`

	some note, tc in {"some expression variables": {
		"declarations": {{"location": "4:7:4:8", "type": "var", "value": "i"}},
		"references": {{"location": "5:2:5:3", "type": "var", "value": "i"}},
	}}

	tokens := some_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	tc.declarations == tokens.declaration
	tc.references == tokens.reference
}
