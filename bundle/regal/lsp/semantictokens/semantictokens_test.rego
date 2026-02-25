package regal.lsp.semantictokens_test

import data.regal.lsp.semantictokens

test_arg_tokens_with_declarations_and_references if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	calc1 := param1 * 2
	calc2 := param2 + 10
	result := calc1 + calc2

	calc3 := 1
	calc3 == param1
}`
	tokens := semantictokens.arg_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	# Check declarations
	{"location": "3:15:3:21", "value": "param1", "type": "var"} in tokens.declaration
	{"location": "3:23:3:29", "value": "param2", "type": "var"} in tokens.declaration

	# Check references
	{"location": "4:11:4:17", "value": "param1", "type": "var"} in tokens.reference
	{"location": "5:11:5:17", "value": "param2", "type": "var"} in tokens.reference
	{"location": "9:11:9:17", "value": "param1", "type": "var"} in tokens.reference
}

test_arg_tokens_declarations_only if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	true
}`
	tokens := semantictokens.arg_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	# Check declarations
	{"location": "3:15:3:21", "value": "param1", "type": "var"} in tokens.declaration
	{"location": "3:23:3:29", "value": "param2", "type": "var"} in tokens.declaration

	# Should have no references since variables aren't used
	count(tokens.reference) == 0
}

test_import_tokens if {
	policy := `package regal.woo

import data.regal.ast

test_function(param1, param2) := result if {
	ast.is_constant
}`
	tokens := semantictokens.import_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	# Check imports
	{"location": "3:19:3:22", "type": "string", "value": "ast"} in tokens
}

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
	tokens := semantictokens.comprehension_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
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
	tokens := semantictokens.every_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

}

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
	tokens := semantictokens.some_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

}

test_arg_tokens_no_variables if {
	policy := `package regal.woo

allow := true`

	tokens := semantictokens.arg_tokens with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	# Should find no tokens for a policy with no function arguments
	count(tokens.declaration) == 0
	count(tokens.reference) == 0
}
