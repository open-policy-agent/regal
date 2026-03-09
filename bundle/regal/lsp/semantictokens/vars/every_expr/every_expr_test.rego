package regal.lsp.semantictokens.vars.every_expr_test

import data.regal.lsp.semantictokens.vars.every_expr

test_every_vars[note] if {
	policy := `package regal.woo
	
every_two_vars if {
	every k, v in input.object {  
		is_string(k)             
		v > 0                    
	}
}`

	some note, tc in {"every expression variables": {
		"declarations": {
			{"location": "4:11:4:12", "type": "var", "value": "v"},
			{"location": "4:8:4:9", "type": "var", "value": "k"},
		},
		"references": {
			{"location": "5:13:5:14", "type": "var", "value": "k"},
			{"location": "6:3:6:4", "type": "var", "value": "v"},
		},
	}}

	tokens := every_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	tc.declarations == tokens.declaration
	tc.references == tokens.reference
}

test_every_single_var_case if {
	policy := `package regal.woo

every_one_var if {
	every k in input.object {  
		is_string(k)                                
	}
}`

	some note, tc in {"every expression variables": {
		"declarations": {{"location": "4:8:4:9", "type": "var", "value": "k"}},
		"references": {{"location": "5:13:5:14", "type": "var", "value": "k"}},
	}}

	tokens := every_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	tc.declarations == tokens.declaration
	tc.references == tokens.reference
}
