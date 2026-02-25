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

import data.regal.ast

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# This is handling the case where the module from the parsed workspace is empty

default result["response"] := {}

# METADATA
# entrypoint: true
result["response"] := {
	"arg_tokens": arg_tokens,
	"package_tokens": package_tokens,
	"import_tokens": import_tokens,
	"comprehension_tokens": comprehension_tokens,
	"every_tokens": every_tokens,
	"some_tokens": some_tokens,
}
