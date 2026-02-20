# METADATA
# description: Use raw strings for regex patterns
package regal.rules.idiomatic["non-raw-regex-pattern"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	# skip traversing refs if no builtin regex function calls are registered
	util.intersects(_re_pattern_function_names, ast.builtin_functions_called)

	value := ast.found.calls[_][_]

	value[0].value[0].value == "regex"

	# The name following "regex.", e.g. "match"
	name := value[0].value[1].value

	some pos in _re_pattern_functions[name]

	value[pos].type == "string"

	loc := util.to_location_object(value[pos].location)
	row := input.regal.file.lines[loc.row - 1]
	chr := substring(row, loc.col - 1, 1)

	chr == `"`

	violation := result.fail(rego.metadata.chain(), result.location(value[pos]))
}

# Mapping of regex.* functions and the position(s)
# of their "pattern" attributes
_re_pattern_functions := {
	"find_all_string_submatch_n": [1],
	"find_n": [1],
	"globs_match": [1, 2],
	"is_valid": [1],
	"match": [1],
	"replace": [2],
	"split": [1],
	"template_match": [1],
}

_re_pattern_function_names := {
	"regex.find_all_string_submatch_n",
	"regex.find_n",
	"regex.globs_match",
	"regex.is_valid",
	"regex.match",
	"regex.replace",
	"regex.split",
	"regex.template_match",
}
