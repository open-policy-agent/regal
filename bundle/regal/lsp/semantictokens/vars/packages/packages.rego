# METADATA
# description: |
#   Helper package for semantictokens that returns packages
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.packages

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract package tokens - return full package path
result contains last_term if {
	package_path := module.package.path

	last_term := package_path[count(package_path) - 1]
}
