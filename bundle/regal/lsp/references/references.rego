# METADATA
# description: implementation of the LSP references feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.context: {type: "object", properties: {includeDeclaration: {type: "boolean"}}}
package regal.lsp.references

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
result.response contains reference if {
	[arg, i] := find.arg_at_position

	some expr in ast.found.expressions[i]

	walk(expr, [_, value])

	value.type == "var"
	value.value == arg.value

	reference := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
	}
}
