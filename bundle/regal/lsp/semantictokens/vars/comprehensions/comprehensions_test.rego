package regal.lsp.semantictokens.vars.comprehensions_test

import data.regal.lsp.semantictokens.vars.comprehensions

test_comprehensions if {
	policy := `package regal.woo

array_comprehensions := [x |  
	some i, x in [1, 2, 3]    
	i == 2                    
]

set_comprehensions := {x |    
	some i, x in [1, 2, 3]    
	i == 2                    
}

object_comprehensions := {k: v |  
	some k, v in [1, 2, 3]       
	v == 2                        
}`
	tokens := comprehensions.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "14:7:14:8", "type": "var", "value": "k"} in tokens.declaration
	{"location": "14:10:14:11", "type": "var", "value": "v"} in tokens.declaration
	{"location": "4:10:4:11", "type": "var", "value": "x"} in tokens.declaration
	{"location": "4:7:4:8", "type": "var", "value": "i"} in tokens.declaration
	{"location": "9:7:9:8", "type": "var", "value": "i"} in tokens.declaration
	{"location": "9:10:9:11", "type": "var", "value": "x"} in tokens.declaration

	{"location": "3:26:3:27", "type": "var", "value": "x"} in tokens.reference
	{"location": "5:2:5:3", "type": "var", "value": "i"} in tokens.reference
	{"location": "8:24:8:25", "type": "var", "value": "x"} in tokens.reference
	{"location": "10:2:10:3", "type": "var", "value": "i"} in tokens.reference
	{"location": "13:27:13:28", "type": "var", "value": "k"} in tokens.reference
	{"location": "13:30:13:31", "type": "var", "value": "v"} in tokens.reference
	{"location": "15:2:15:3", "type": "var", "value": "v"} in tokens.reference
}
