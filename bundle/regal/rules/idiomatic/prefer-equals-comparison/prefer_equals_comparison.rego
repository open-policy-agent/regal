# METADATA
# description: Prefer `==` for equality comparison
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-equals-comparison
package regal.rules.idiomatic["prefer-equals-comparison"]

import future.keywords.not
import future.keywords.or

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index, expr
	ast.found.expressions[rule_index][expr].terms[0].value[0].value == "eq"
	expr.terms[0].type == "ref"

	_unassignable(expr.terms[1], rule_index)
	_unassignable(expr.terms[2], rule_index)

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

_unassignable(term, rule_index) if {
	ast.is_constant(term)
} or {
	term.type == "ref"
	ast.static_ref(term.value)
} or {
	term.type == "var"
	not ast.is_output_var(rule_index, term)
	not {
		some comp_term in _comprehension_term_vars_in_scope(rule_index, term.location)
		comp_term.value == term.value
		comp_term.type == "var"
	}
}

_comprehension_term_vars_in_scope(rule_index, location) := {node |
	loc := util.to_location_object(location)

	some comp in ast.found.comprehensions[rule_index]
	util.contains_location(util.to_location_object(comp.location), loc)

	fields := {
		"arraycomprehension": ["term"],
		"objectcomprehension": ["key", "value"],
		"setcomprehension": ["term"],
	}[comp.type]

	some field in fields
	term := comp.value[field]

	walk(term, [_, node])

	node.type == "var"
}
