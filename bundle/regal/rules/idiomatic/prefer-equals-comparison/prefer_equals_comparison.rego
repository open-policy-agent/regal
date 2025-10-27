# METADATA
# description: Prefer `==` for equality comparison
package regal.rules.idiomatic["prefer-equals-comparison"]

import data.regal.ast
import data.regal.result

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
	not ast.is_output_var(input.rules[to_number(rule_index)], term)
}
