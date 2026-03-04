package regal.lsp.semantictokens.vars.comprehensions_test

import data.regal.lsp.semantictokens.vars.comprehensions

test_array_comprehension[note] if {
	policy_one := `package regal.woo

array_comprehensions := [x |  
	some i, x in [1, 2, 3]    
	i == 2                    
]`
	some note, tc in {"array comprehensions": {
		"declarations": {
			{"location": "4:10:4:11", "type": "var", "value": "x"},
			{"location": "4:7:4:8", "type": "var", "value": "i"},
		},
		"references": {
			{"location": "3:26:3:27", "type": "var", "value": "x"},
			{"location": "5:2:5:3", "type": "var", "value": "i"},
		},
	}}
	array_comp_tokens := comprehensions.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy_one)

	tc.declarations == array_comp_tokens.declaration
	tc.references == array_comp_tokens.reference
}

test_set_comprehensions[note] if {
	policy_one := `package regal.woo

set_comprehensions := {x |    
	some i, x in [1, 2, 3]    
	i == 2                    
}`
	some note, tc in {"set comprehensions": {
		"declarations": {
			{"location": "4:10:4:11", "type": "var", "value": "x"},
			{"location": "4:7:4:8", "type": "var", "value": "i"},
		},
		"references": {
			{"location": "3:24:3:25", "type": "var", "value": "x"},
			{"location": "5:2:5:3", "type": "var", "value": "i"},
		},
	}}
	set_comp_tokens := comprehensions.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy_one)

	tc.declarations == set_comp_tokens.declaration
	tc.references == set_comp_tokens.reference
}

test_object_comprehension[note] if {
	policy_one := `package regal.woo

object_comprehensions := {k: v |  
	some k, v in [1, 2, 3]       
	v == 2                        
}`
	some note, tc in {"object comprehensions": {
		"declarations": {
			{"location": "4:10:4:11", "type": "var", "value": "v"},
			{"location": "4:7:4:8", "type": "var", "value": "k"},
		},
		"references": {
			{"location": "3:27:3:28", "type": "var", "value": "k"},
			{"location": "3:30:3:31", "type": "var", "value": "v"},
			{"location": "5:2:5:3", "type": "var", "value": "v"},
		},
	}}
	object_comp_tokens := comprehensions.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy_one)

	tc.declarations == object_comp_tokens.declaration
	tc.references == object_comp_tokens.reference
}
