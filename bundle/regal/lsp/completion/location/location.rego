# METADATA
# description: various rules and functions related to location and position
package regal.lsp.completion.location

import data.regal.ast
import data.regal.util

# METADATA
# description: best-effort helper to determine if the current line is in a rule body
# scope: document
in_rule_body(line) if contains(line, " if ")
in_rule_body(line) if contains(line, " contains ")
in_rule_body(line) if contains(line, " else ")
in_rule_body(line) if contains(line, "= ")
in_rule_body(line) if regex.match(`^\s+`, line)

# METADATA
# description: converts OPA location to LSP position
to_position(location) := {
	"line": location.row - 1,
	"character": location.col - 1,
}

# METADATA
# description: returns a range from start of line to position
from_start_of_line_to_position(position) := {
	"start": {
		"line": position.line,
		"character": 0,
	},
	"end": {
		"line": position.line,
		"character": position.character,
	},
}

# METADATA
# description: |
#   estimate where the location "ends" based on its text attribute,
#   both line and column
end_location_estimate(location) := {
	"row": location.row + strings.count(location.text, "\n"),
	"col": count(regex.replace(location.text, `.*\n`, "")),
}

# METADATA
# description: |
#   find and return rule at provided row
#   undefined if provided row is not within the range of a rule
find_rule(rules, row) := [rule |
	some i, rule in rules

	start_location := util.to_location_object(rule.location)

	row >= start_location.row
	row <= end_location_estimate(start_location).row
][0]

# METADATA
# description: |
#   find local variables (declared via function arguments, some/every
#   declarations or assignment) at the given location. note that this expects
#   `location` as a map, not a string
find_locals(rules, location) := ast.find_names_in_local_scope(find_rule(rules, location.row), location)

# METADATA
# description: |
#   return the range for a word object (as return by `word_at`, or `ref_at`)
word_range(word, position) := {
	"start": {
		"line": position.line,
		"character": position.character - word.offset_before,
	},
	"end": {
		"line": position.line,
		"character": position.character + word.offset_after,
	},
}

# METADATA
# description: |
#   find word at column in line, and return its text, and the offset
#   and text from the position (before and after)
word_at(line, col) := word if {
	text_before := substring(line, 0, col - 1)
	word_before := _to_string(regex.find_n(`[a-zA-Z_][a-zA-Z0-9_]*$|:$`, text_before, 1))

	text_after := trim_prefix(line, text_before)
	word_after := _to_string(regex.find_n(`^[a-zA-Z0-9_]+|^:`, text_after, 1))

	offset_before := count(word_before)
	offset_after := count(word_after)

	word := {
		"text_before": substring(text_before, 0, count(text_before) - offset_before),
		"text_after": substring(text_after, offset_after, count(text_after)),
		"offset_before": offset_before,
		"offset_after": offset_after,
		"text": $"{word_before}{word_after}",
	}
}

# METADATA
# description: |
#   find ref at column in line, and return its text, and the offset
#   from the position (before and after)
#   this is similar to word_at but captures `.` as well
ref_at(line, col) := word if {
	text_before := substring(line, 0, col - 1)
	word_before := _to_string(regex.find_n(`[a-zA-Z_][a-zA-Z0-9_\.]*$`, text_before, 1))

	text_after := substring(line, col - 1, count(line))
	word_after := _to_string(regex.find_n(`^[a-zA-Z0-9_\.]+`, text_after, 1))

	word := {
		"offset_before": count(word_before),
		"offset_after": count(word_after),
		"text": $"{word_before}{word_after}",
	}
}

_to_string(arr) := "" if count(arr) == 0
_to_string(arr) := arr[0] if count(arr) > 0
