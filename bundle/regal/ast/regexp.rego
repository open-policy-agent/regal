package regal.ast

import data.regal.util

# METADATA
# description: All regular expression 'literals' (represented as strings) found in the AST
regexp.found.literals contains term if {
	# skip traversing refs if no builtin regex function calls are registered
	util.intersects(regexp.pattern_function_names, builtin_functions_called)

	value := found.calls[_][_]

	value[0].value[0].value == "regex"

	# The name following "regex.", e.g. "match"
	name := value[0].value[1].value

	some pos in regexp.pattern_functions[name]

	term := value[pos]

	term.type == "string"
}

# METADATA
# description: Mapping of regex.* functions and the position(s) of their "pattern" attributes
regexp.pattern_functions := {
	"find_all_string_submatch_n": [1],
	"find_n": [1],
	"globs_match": [1, 2],
	"is_valid": [1],
	"match": [1],
	"replace": [2],
	"split": [1],
	"template_match": [1],
}

# METADATA
# description: Set of all regex.* function names that take a regex pattern as an argument
regexp.pattern_function_names := {
	"regex.find_all_string_submatch_n",
	"regex.find_n",
	"regex.globs_match",
	"regex.is_valid",
	"regex.match",
	"regex.replace",
	"regex.split",
	"regex.template_match",
}
