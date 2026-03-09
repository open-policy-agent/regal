package regal.lsp.semantictokens.vars.function_args_test

import data.regal.lsp.semantictokens.vars.function_args

test_function_args_with_declarations_and_references if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	calc1 := param1 * 2
	calc2 := param2 + 10
	result := calc1 + calc2

	calc3 := 1
	calc3 == param1
}`
	tokens := function_args.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "3:15:3:21", "value": "param1", "type": "var"} in tokens.declaration
	{"location": "3:23:3:29", "value": "param2", "type": "var"} in tokens.declaration

	{"location": "4:11:4:17", "value": "param1", "type": "var"} in tokens.reference
	{"location": "5:11:5:17", "value": "param2", "type": "var"} in tokens.reference
	{"location": "9:11:9:17", "value": "param1", "type": "var"} in tokens.reference
}

test_function_args_declarations_only if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	true
}`
	tokens := function_args.result with input as {"params": {"textDocument": {"uri": "file://p.rego"}}}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	{"location": "3:15:3:21", "value": "param1", "type": "var"} in tokens.declaration
	{"location": "3:23:3:29", "value": "param2", "type": "var"} in tokens.declaration

	# Should have no references since variables aren't used
	count(tokens.reference) == 0
}
