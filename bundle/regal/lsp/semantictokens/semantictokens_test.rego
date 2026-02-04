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

	module := regal.parse_module("policy.rego", policy)

	tokens := semantictokens.arg_tokens with input as module

	tokens[{"location": "3:15:3:21", "value": "param1", "type": "var"}] == "declaration"
	tokens[{"location": "3:23:3:29", "value": "param2", "type": "var"}] == "declaration"

	tokens[{"location": "4:11:4:17", "value": "param1", "type": "var"}] == "reference"
	tokens[{"location": "5:11:5:17", "value": "param2", "type": "var"}] == "reference"
	tokens[{"location": "9:11:9:17", "value": "param1", "type": "var"}] == "reference"
}

test_arg_tokens_declarations_only if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	true
}`

	module := regal.parse_module("policy.rego", policy)

	tokens := semantictokens.arg_tokens with input as module

	tokens[{"location": "3:15:3:21", "value": "param1", "type": "var"}] == "declaration"
	tokens[{"location": "3:23:3:29", "value": "param2", "type": "var"}] == "declaration"
}

test_arg_tokens_no_variables if {
	policy := `package regal.woo

allow := true`

	module := regal.parse_module("policy.rego", policy)

	tokens := semantictokens.arg_tokens with input as module

	# Should find no tokens for a policy with no function arguments
	count(tokens) == 0
}
