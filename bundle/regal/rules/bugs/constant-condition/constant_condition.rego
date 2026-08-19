# METADATA
# description: Constant condition
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition
package regal.rules.bugs["constant-condition"]

import data.regal.ast
import data.regal.result

# METADATA
# description: single scalar value or templatestring, like a lone `true` inside a rule body
# scope: rule
report contains violation if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	# `and`/`or` operands are excluded, as removing the expression would leave
	# the enclosing expression without an operand
	not ast.logical_operand_locations[rule_index][expr.location]

	terms := expr.terms

	# We could include composite types too, but less comomon and more expensive to check
	terms.type in _scalar_expr_types

	violation := result.fail(rego.metadata.chain(), result.location(terms))
}

# METADATA
# description: two scalar values with a "boolean operator" between, like 1 == 1, or 2 > 1
# scope: rule
report contains violation if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	not ast.logical_operand_locations[rule_index][expr.location]

	_comparison_locations[rule_index][expr.location]

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

# METADATA
# description: '`and`/`or` expression where every operand is constant, like `1 or 2`'
# scope: rule
report contains violation if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in _logical_expr_types

	# only the outermost logical expression is reported, as removing a nested
	# one would leave its enclosing expression without an operand
	not ast.logical_operand_locations[rule_index][expr.location]

	constant_locations := _constant_expr_locations[rule_index]

	# every `and`/`or` node in the expression contributes its own operands here, so
	# nested logical expressions have their operands checked as well
	every operand in [operand |
		walk(expr.terms, [_, node])

		node.type in _logical_expr_types

		some operand in [node.lhs, node.rhs]
	] {
		count(operand) == 1
		operand[0].location in constant_locations
	}

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

_scalar_expr_types := {"boolean", "null", "number", "string", "templatestring"}

_logical_expr_types := {"and", "or"}

# METADATA
# description: |
#   locations of the expressions comparing two scalar values with a "boolean
#   operator" between them, like `1 == 1`, in the rule at rule_index
_comparison_locations[rule_index] contains expr.location if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms[0].value[0].type == "var"
	expr.terms[0].value[0].value in {"equal", "gt", "gte", "lt", "lte", "neq"}

	expr.terms[1].type in ast.scalar_types
	expr.terms[2].type in ast.scalar_types
}

# METADATA
# description: |
#   locations of the expressions that are constant on their own, in the rule at rule_index.
#   `and`/`or` expressions are included, as whether they are constant is determined by their
#   operands, which are checked separately
_constant_expr_locations[rule_index] contains expr.location if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in (_scalar_expr_types | _logical_expr_types)
}

_constant_expr_locations[rule_index] contains location if {
	some rule_index, locations in _comparison_locations

	some location in locations
}
