# METADATA
# description: |
#   Helper package for semantictokens that returns packages
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.semantictokens.vars.packages

import data.regal.util

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Package keyword
result contains token if {
	tloc := util.to_location_object(module.package.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 3,
		"modifiers": 0,
	}
}

# METADATA
# description: Last package path term as token
result contains token if {
	term := regal.last(module.package.path)
	tloc := util.to_location_object(term.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 0,
		"modifiers": 0,
	}
}
