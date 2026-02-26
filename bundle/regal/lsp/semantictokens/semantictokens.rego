# METADATA
# description: |
#   Returns location of variables to be highlighted via semantic tokens. Currently returns:
#     - declarations of function args in text documents
#     - variable references that are used in function calls
#     - variable references that are used in expressions
#     - variables declarations and references in comprehensions
#     - variables declarations and references in every/some keyword domains
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens

import data.regal.lsp.semantictokens.vars.comprehensions
import data.regal.lsp.semantictokens.vars.every_expr
import data.regal.lsp.semantictokens.vars.function_args
import data.regal.lsp.semantictokens.vars.imports
import data.regal.lsp.semantictokens.vars.packages
import data.regal.lsp.semantictokens.vars.some_expr

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# This is handling the case where the module from the parsed workspace is empty
default result["response"] := {}

# METADATA
# entrypoint: true
result["response"] := {
	"packages": packages.result,
	"imports": imports.result,
	"vars": {
		"function_args": function_args.result,
		"comprehensions": comprehensions.result,
		"every_expr": every_expr.result,
		"some_expr": some_expr.result,
	},
}
