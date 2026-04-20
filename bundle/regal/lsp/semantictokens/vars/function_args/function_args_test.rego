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
	tokens := function_args.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 14, "length": 6, "line": 2, "modifiers": 1, "type": 1},
		{"col": 22, "length": 6, "line": 2, "modifiers": 1, "type": 1},
		{"col": 10, "length": 6, "line": 3, "modifiers": 2, "type": 1},
		{"col": 10, "length": 6, "line": 4, "modifiers": 2, "type": 1},
		{"col": 10, "length": 6, "line": 8, "modifiers": 2, "type": 1},
	}
}

test_function_args_declarations_only if {
	policy := `package regal.woo

test_function(param1, param2) := result if {
	true
}`
	tokens := function_args.result with data.workspace.parsed["file:///p.rego"] as regal.parse_module("p.rego", policy)
		with input.params.textDocument.uri as "file:///p.rego"
		with input.regal.file.lines as split(policy, "\n")

	tokens == {
		{"col": 14, "length": 6, "line": 2, "modifiers": 1, "type": 1},
		{"col": 22, "length": 6, "line": 2, "modifiers": 1, "type": 1},
	}
}
