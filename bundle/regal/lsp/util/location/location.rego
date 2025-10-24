# METADATA
# description: utility functions for dealing with location data in the LSP
package regal.lsp.util.location

# METADATA
# description: turns an AST location (with `end`` attribute) into an LSP range
to_range(location) := {
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
#   parse location string sl to LSP Range object
#   example:
#   "1:5:1:10" -> {"start": {"line":0,"character":4},"end":{"line":0,"character":9}}
parse_range(sl) := {
	"start": {
		"line": to_number(r) - 1,
		"character": to_number(c) - 1,
	},
	"end": {
		"line": to_number(er) - 1,
		"character": to_number(ec) - 1,
	},
} if {
	[r, c, er, ec] := split(sl, ":")
}

# METADATA
# description: checks if a given position 'pos' is within range 'rng'
# scope: document
within_range(pos, rng) if {
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
