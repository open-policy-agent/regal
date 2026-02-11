# METADATA
# description: |
#   Returns location of variables to be highlighted via semantic tokens. Currently returns:
#     - declarations of function args in text documents
#     - variable references that are used in function calls
#     - variable references that are used in expressions
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# entrypoint: true
result.response := {
	"arg_tokens": arg_tokens,
	"package_tokens": package_tokens,
}

# METADATA
# description: Extract function argument declarations
arg_tokens.declaration contains var if {
	some rule in module.rules
	some var in rule.head.args
}

# METADATA
# description: Extract variable references in function calls
arg_tokens.reference contains var if {
	some rule in module.rules
	walk(rule.body, [_, expr])
	expr.terms[0].type == "ref"
	some var in array.slice(expr.terms, 1, count(expr.terms))
	var.type == "var"

	arg_names := {v.value | some v in rule.head.args}
	var.value in arg_names
}

# METADATA
# description: Extract variable references in call expressions
arg_tokens.reference contains var if {
	some rule in module.rules
	walk(rule.body, [_, expr])

	some term in expr.terms
	term.type == "call"

	some var in term.value
	var.type == "var"

	arg_names := {v.value | some v in rule.head.args}
	var.value in arg_names
}

# METADATA
# description: Extract package tokens - return full package path
package_tokens := module.package.path
