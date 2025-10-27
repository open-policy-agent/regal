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
		"line": to_number(r) - 1,
		"character": to_number(c) - 1,
	},
	"end": {
		"line": to_number(er) - 1,
		"character": to_number(ec) - 1,
	},
} if {
	[r, c, er, ec] := split(location_string, ":")
}

# METADATA
# description: checks if a given position 'pos' is found within range 'rng'
# scope: document
contains_position(rng, pos) if {
	pos.line > rng.start.line
	pos.line < rng.end.line
} else if {
	pos.line == rng.start.line
	pos.line < rng.end.line
	pos.character >= rng.start.character
} else if {
	pos.line > rng.start.line
	pos.line == rng.end.line
	pos.character <= rng.end.character
} else if {
	pos.line == rng.start.line
	pos.line == rng.end.line
	pos.character >= rng.start.character
	pos.character <= rng.end.character
}
