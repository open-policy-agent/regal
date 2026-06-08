# METADATA
# description: implementation of the LSP prepare rename feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
package regal.lsp.preparerename

import data.regal.lsp.util.find
import data.regal.lsp.util.range

default result["response"] := null

# METADATA
# entrypoint: true
result["response"] := response if {
	[arg, _] := find.arg_at_position

	response := {
		"placeholder": arg.value,
		"range": range.parse(arg.location),
	}
}
