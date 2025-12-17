# METADATA
# description: Constant condition
package regal.rules.bugs["constant-condition"]

import data.regal.ast
import data.regal.result

# METADATA
# description: single scalar value or templatestring, like a lone `true` inside a rule body
# scope: rule
report contains violation if {
	terms := ast.found.expressions[_][_].terms

	# We could include composite types too, but less comomon and more expensive to check
	terms.type in {"boolean", "null", "number", "string", "templatestring"}

	violation := result.fail(rego.metadata.chain(), result.location(terms))
}

# METADATA
# description: two scalar values with a "boolean operator" between, like 1 == 1, or 2 > 1
# scope: rule
report contains violation if {
	operators := {"equal", "gt", "gte", "lt", "lte", "neq"}

	expr := ast.found.expressions[_][_]

	expr.terms[0].value[0].type == "var"
	expr.terms[0].value[0].value in operators

	expr.terms[1].type in ast.scalar_types
	expr.terms[2].type in ast.scalar_types

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}
