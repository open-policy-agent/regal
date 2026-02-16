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
	"import_tokens": import_tokens,
}

# METADATA
# description: Extract import tokens - return only last term of the path
import_tokens contains last_term if {
	some import_statement in module.imports
	import_path := import_statement.path.value

	last_term := import_path[count(import_path) - 1]
}

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

# METADATA
# description: Extract package tokens - return full package path
package_tokens := module.package.path
