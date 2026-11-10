# METADATA
# description: implementation of the LSP rename feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.newName: {type: "string"}
package regal.lsp.rename

import data.regal.lsp.util.references

# METADATA
# entrypoint: true
default result.response := null

result.response := {"changes": _changes} if _changes != {}

_changes[ref.uri] contains change if {
	some ref in references.refs

	change := {
		"range": ref.range,
		"newText": input.params.newName,
	}
}
