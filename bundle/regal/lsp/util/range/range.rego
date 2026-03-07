# METADATA
# description: |
#   Utilities for working with Range data from the Language Server Protocol (LSP), and
#   conversions to/from AST location formats.
# related_resources:
#   - description: LSP Range Specification
#     ref: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#range
package regal.lsp.util.range

# METADATA
# description: |
#   turns a parsed AST location (with `end`` attribute present) into an LSP range,
#   assuming a valid location is provided and without doing any validation thereof
from_location(location) := {
	"start": {
		"line": location.row - 1,
		"character": location.col - 1,
	},
	"end": {
		"line": location.end.row - 1,
		"character": location.end.col - 1,
	},
}

# METADATA
# description: |
#   parse AST location_string (columns and rows staring from 1)
#   to LSP Range object (lines and characters starting from 0)
#   example:
#   "1:5:1:10" -> {"start": {"line":0, "character": 4}, "end":{"line": 0,"character": 9}}
# related_resources:
#   - description: RoAST compact location format
#     ref: https://www.openpolicyagent.org/projects/regal/custom-rules/roast#compact-location-format
parse(location_string) := {
	"start": {
		"line": to_number(parts[0]) - 1,
		"character": to_number(parts[1]) - 1,
	},
	"end": {
		"line": to_number(parts[2]) - 1,
		"character": to_number(parts[3]) - 1,
	},
} if {
	parts := split(location_string, ":")
}

# METADATA
# description: checks if a given position 'pos' is found within 'range'
# scope: document
contains_position(range, pos) if {
	pos.line > range.start.line
	pos.line < range.end.line
} else if {
	pos.line == range.start.line
	pos.line < range.end.line
	pos.character >= range.start.character
} else if {
	pos.line > range.start.line
	pos.line == range.end.line
	pos.character <= range.end.character
} else if {
	pos.line == range.start.line
	pos.line == range.end.line
	pos.character >= range.start.character
	pos.character <= range.end.character
}
