# METADATA
# description: Iteration in top-level assignment
package regal.rules.bugs["top-level-iteration"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i
	input.rules[i].head.value.type == "ref"
	rule := input.rules[i]

	# skip if vars in the ref head
	every term in array.slice(rule.head.ref, 1, 100) {
		term.type != "var"
	}

	some term in array.slice(rule.head.value.value, 1, 100)

	term.type == "var"

	not term.value in ast.identifiers
	not _is_arg_or_input(term.value, rule)

	# this is expensive, but the preconditions should ensure that
	# very few rules evaluate this far
	not _var_in_body(rule, term.value)

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

_var_in_body(rule, value) if {
	walk(rule.body, [_, node])
	node.type == "var"
	node.value == value
}

_is_arg_or_input(value, rule) if value in ast.function_arg_names(rule)
_is_arg_or_input(value, _) if value[0].value == "input"
_is_arg_or_input("input", _)
