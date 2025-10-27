# METADATA
# description: Pointless reassignment of variable
package regal.rules.style["pointless-reassignment"]

import data.regal.ast
import data.regal.result

# pointless reassignment in rule head
report contains violation if {
	some rule in ast.rules

	not rule.body

	rule.head.value.type == "var"
	count(rule.head.ref) == 1

	violation := result.fail(rego.metadata.chain(), result.location(rule))
}

# pointless reassignment in expressions
report contains violation if {
	some expr
	ast.found.expressions[_][expr].terms[0].value[0].value == "assign"

	not expr.with
	expr.terms[0].type == "ref"
	expr.terms[1].type == "var"
	expr.terms[2].type == "var"

	violation := result.fail(rego.metadata.chain(), result.infix_expr_location(expr.terms))
}
