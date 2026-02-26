package regal.lsp.semantictokens.vars.every_expr_test

import data.regal.lsp.semantictokens.vars.every_expr

test_every_vars if {
	policy := `package regal.woo
	
every_two_vars_construct if {
	every k, v in input.object {  
		is_string(k)             
		v > 0                    
	}
}

every_one_var_construct if {
	every k in input.object {  
		is_string(k)                                
	}
}`
	tokens := every_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "11:8:11:9", "type": "var", "value": "k"} in tokens.declaration
	{"location": "4:11:4:12", "type": "var", "value": "v"} in tokens.declaration
	{"location": "4:8:4:9", "type": "var", "value": "k"} in tokens.declaration

	{"location": "12:13:12:14", "type": "var", "value": "k"} in tokens.reference
	{"location": "5:13:5:14", "type": "var", "value": "k"} in tokens.reference
	{"location": "6:3:6:4", "type": "var", "value": "v"} in tokens.reference
}
