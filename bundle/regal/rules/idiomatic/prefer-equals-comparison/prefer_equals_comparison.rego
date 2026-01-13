# METADATA
# description: Prefer `==` for equality comparison
package regal.rules.idiomatic["prefer-equals-comparison"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index, expr
	ast.found.expressions[rule_index][expr].terms[0].value[0].value == "eq"
	expr.terms[0].value[0].type == "var"
	expr.terms[0].type == "ref"

	_unassignable(expr.terms[1], rule_index)
	_unassignable(expr.terms[2], rule_index)

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

_unassignable(term, _) if ast.is_constant(term)

_unassignable(term, _) if {
	term.type == "ref"
	ast.static_ref(term.value)
}

_unassignable(term, rule_index) if {
	term.type == "var"
	ri := to_number(rule_index)
	not ast.is_output_var(input.rules[ri], term)
	not _is_declared_comp_term(term, rule_index)
}

_is_declared_comp_term(term, rule_index) if {
	some comp_term in _comprehension_term_vars_in_scope(rule_index, term.location)
	comp_term.type == "var"
	comp_term.value == term.value
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
