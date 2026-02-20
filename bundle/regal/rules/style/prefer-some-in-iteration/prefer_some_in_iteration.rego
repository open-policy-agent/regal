# METADATA
# description: Prefer `some .. in` for iteration
package regal.rules.style["prefer-some-in-iteration"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	cfg := config.rules.style["prefer-some-in-iteration"]
	some i, rule_index in ast.rule_index_strings

	rule := input.rules[i]

	not _possible_top_level_iteration(rule)

	some ref in ast.found.refs[rule_index]

	vars_in_ref := [term |
		some term in array.slice(ref.value, 1, 100)

		term.type == "var"
	]
	vars_in_ref != []

	# we don't need the location of each var, but using the first
	# ref will do, and will hopefully help with caching the result
	num_output_vars := count([ast.is_output_var(rule, var) | some var in vars_in_ref])
	num_output_vars != 0
	num_output_vars < cfg["ignore-nesting-level"]

	not _except_sub_attribute(cfg, ref.value)
	not _invalid_some_location(rule, ref.location)

	violation := result.fail(rego.metadata.chain(), result.location(ref))
}

_except_sub_attribute(cfg, ref) if {
	cfg["ignore-if-sub-attribute"] == true
	_has_sub_attribute(ref)
}

_has_sub_attribute(terms) if {
	last_var_pos := regal.last([i |
		some i, term in terms
		term.type == "var"
	])
	last_var_pos < count(terms) - 1
}

# don't walk top level iteration refs:
# https://www.openpolicyagent.org/projects/regal/rules/bugs/top-level-iteration
_possible_top_level_iteration(rule) if {
	not rule.body
	rule.head.value.type == "ref"
}

_invalid_some_location(rule, location) if {
	some node in ["head", "body", "else"]
	walk(rule[node], [path, value])

	value.location == location

	_invalid_some_context(rule, array.flatten([node, path]))
}

# don't recommend `some .. in` if iteration occurs inside of arrays, objects, or sets
_invalid_some_context(rule, path) if {
	some p in util.all_paths(path)

	node := object.get(rule, p, false)

	_impossible_some(node)
}

# don't recommend `some .. in` if iteration occurs inside of a
# function call args list, like `startswith(input.foo[_], "foo")`
# this should honestly be a rule of its own, I think, but it's
# not _directly_ replaceable by `some .. in`, so we'll leave it
# be here
_invalid_some_context(rule, path) if {
	some p in util.all_paths(path)

	node := object.get(rule, p, [])

	node.terms[0].type == "ref"
	node.terms[0].value[0].type == "var"
	node.terms[0].value[0].value in ast.all_function_names
	not node.terms[0].value[0].value in ast.operators
}

# if previous node is of type call, also don't recommend `some .. in`
_invalid_some_context(rule, path) if object.get(rule, array.slice(path, 0, count(path) - 2), {}).type == "call"

_impossible_some(node) if node.type in {"array", "object", "set"}
_impossible_some(node) if node.key

# technically this is not an _impossible_ some, as we could replace e.g. `"x" == input[_]`
# with `some "x" in input`, but that'd be an `unnecessary-some` violation as `"x" in input`
# would be the correct way to express that
_impossible_some(node) if {
	node.terms[0].value[0].type == "var"
	node.terms[0].value[0].value in {"eq", "equal"}

	some term in node.terms
	term.type in ast.scalar_types
}
