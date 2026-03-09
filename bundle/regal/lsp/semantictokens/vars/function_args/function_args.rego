# METADATA
# description: |
#   Helper package for semantictokens that returns function argument references and declarations
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.function_args

import data.regal.ast

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract function argument declarations
result.declaration contains arg if {
	some rule in module.rules
	some arg in rule.head.args
	arg.type == "var"
}

# METADATA
# description: Extract variable references in function calls
result.reference contains arg if {
	some rule in module.rules
	some rule_index

	rule.head.args

	arg_names := ast.function_arg_names(rule)

	call := ast.found.calls[rule_index][_]
	some arg in array.slice(call, 1, 100)

	arg.type == "var"
	arg.value in arg_names
}

# METADATA
# description: Extract variable references in call expressions
result.reference contains arg if {
	some rule in module.rules
	arg_names := ast.function_arg_names(rule)
	walk(rule.body, [_, expr])

	some term in expr.terms
	term.type == "call"

	some arg in term.value
	arg.type == "var"

	arg.value in arg_names
}
