package regal.lsp.semantictokens

# METADATA
# description: Extract function argument declarations
arg_tokens.declaration contains arg if {
	some rule in module.rules
	some arg in rule.head.args
	arg.type == "var"
}

# METADATA
# description: Extract variable references in function calls
arg_tokens.reference contains arg if {
	some rule in module.rules

	rule.head.args

	arg_names := {v.value | some v in rule.head.args}

	walk(rule.body, [_, expr])

	expr.terms[0].type == "ref"

	some arg in array.slice(expr.terms, 1, count(expr.terms))

	arg.type == "var"
	arg.value in arg_names
}

# METADATA
# description: Extract variable references in call expressions
arg_tokens.reference contains arg if {
	some rule in module.rules
	arg_names := {v.value | some v in rule.head.args}
	walk(rule.body, [_, expr])

	some term in expr.terms
	term.type == "call"

	some arg in term.value
	arg.type == "var"

	arg.value in arg_names
}
