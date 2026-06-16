# METADATA
# description: implementation of the LSP prepare rename feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
package regal.lsp.preparerename

import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := _response_for(arg) if [arg, _] := find.arg_at_position

result["response"] := _response_for(var) if [var, _] := find.some_var_at_position

_response_for(term) := {
	"placeholder": term.value,
	"range": range.parse(term.location),
}
