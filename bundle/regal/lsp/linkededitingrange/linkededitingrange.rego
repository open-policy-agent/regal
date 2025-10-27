# METADATA
# description: |
#   Linked editing ranges allow having a local rename *on type* reflect in multiple places. This arguably a potentialy
#   confusing feature, and seems to be off by default in at least VS Code, so unless we find strong use-cases for it, we
#   should not invest much effort into polishing this. Currently we only provide experimental support of linked editing
#   ranges for:
#     - function arguments and references to them in the function head and body
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_linkedEditingRange
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.linkededitingrange

import data.regal.ast
import data.regal.util

import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
default result.response.ranges := set()

# This is currently an experimental feature kept behind a flag to not accidedntally result in a poor
# user experience for anyone who perhaps mistakenly enabled linked editing in their editor. Handler, and
# the code kept here, to allow us to quickly test out the feature for different use-cases later.
result.response.ranges := ranges if util.parse_bool(opa.runtime().env.REGAL_EXPERIMENTAL)

# METADATA
# description: Link a function args in position
ranges contains range.parse(arg.location) if [arg, _] := find.arg_at_position

# METADATA
# description: Link function arg references in function body to arg
ranges contains range.parse(value.location) if {
	[arg, i] := find.arg_at_position

	some expr in ast.found.expressions[sprintf("%d", [i])]

	walk(expr, [_, value])

	value.type == "var"
	value.value == arg.value
}

# METADATA
# description: Link function arg references in head value to arg
ranges contains range.parse(value.location) if {
	[arg, i] := find.arg_at_position

	walk(data.workspace.parsed[input.params.textDocument.uri].rules[i].head.value, [_, value])

	value.type == "var"
	value.value == arg.value
}
