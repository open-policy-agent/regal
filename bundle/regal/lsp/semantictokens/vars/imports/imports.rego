# METADATA
# description: |
#   Helper package for semantictokens that returns imports
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.imports

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract import tokens - return only last term of the path
result contains last_term if {
	some import_statement in module.imports
	import_path := import_statement.path.value

	last_term := import_path[count(import_path) - 1]
}
