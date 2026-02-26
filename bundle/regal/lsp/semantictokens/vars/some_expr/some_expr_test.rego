package regal.lsp.semantictokens.vars.some_expr_test

import data.regal.lsp.semantictokens.vars.some_expr

test_some_vars if {
	policy := `package regal.woo

some_two_vars_construct if {
	some i, item in input.array   
	i < 10                        
	item > 0                        
}

some_one_var_construct if {
	some i in input.array   
	i < 10                                              
}`
	tokens := some_expr.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "10:7:10:8", "type": "var", "value": "i"} in tokens.declaration
	{"location": "4:10:4:14", "type": "var", "value": "item"} in tokens.declaration
	{"location": "4:7:4:8", "type": "var", "value": "i"} in tokens.declaration

	{"location": "11:2:11:3", "type": "var", "value": "i"} in tokens.reference
	{"location": "5:2:5:3", "type": "var", "value": "i"} in tokens.reference
	{"location": "6:2:6:6", "type": "var", "value": "item"} in tokens.reference
}
