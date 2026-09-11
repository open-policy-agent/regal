# METADATA
# description: Constant condition
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition
package regal.rules.bugs["constant-condition"]

import data.regal.ast
import data.regal.result
import data.regal.util

# METADATA
# description: single scalar value or templatestring, like a lone `true` inside a rule body
# scope: rule
report contains violation if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	# `and`/`or` operands are excluded, as removing the expression would leave the enclosing
	# expression without an operand — they're reported by the rules below instead
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

	# only the outermost logical expression is reported here, as a nested one is an
	# operand of its enclosing expression, and reported as such by the rule below
	not ast.logical_operand_locations[rule_index][expr.location]

	expr.location in _constant_logical_expr_locations[rule_index]

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

# METADATA
# description: '`and`/`or` expression with a constant operand, like the `1` in `input.a and 1`'
# scope: rule
report contains violation if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in _logical_expr_types

	some side in ["lhs", "rhs"]

	other_side := _other_side[side]

	_constant_operand(rule_index, expr.terms[side])

	# expressions where both operands are constant are reported by the rule above,
	# as those are removed rather than collapsed
	not _constant_operand(rule_index, expr.terms[other_side])

	# the expression can only be collapsed to the operand kept if that operand may
	# stand on its own, so any other constant operand is left alone
	_stands_alone(expr.terms, other_side)

	violation := result.fail(rego.metadata.chain(), result.location(_collapse_location(expr, side)))
}

_scalar_expr_types := {"boolean", "null", "number", "string", "templatestring"}

_logical_expr_types := {"and", "or"}

_other_side := {"lhs": "rhs", "rhs": "lhs"}

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
#   locations of the expressions that are constant on their own, in the rule at rule_index
_constant_expr_locations[rule_index] contains expr.location if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in _scalar_expr_types
}

_constant_expr_locations[rule_index] contains location if {
	some rule_index, locations in _comparison_locations

	some location in locations
}

# METADATA
# description: locations of the `and`/`or` expressions in the rule at rule_index
_logical_expr_locations[rule_index] contains expr.location if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in _logical_expr_types
}

# METADATA
# description: |
#   locations of the `and`/`or` expressions where every operand is constant, like `1 or 2`,
#   in the rule at rule_index. Unlike the rule reporting them, nested expressions are
#   included here, as an enclosing expression is only constant if they are
_constant_logical_expr_locations[rule_index] contains expr.location if {
	some rule_index, i

	expr := ast.found.expressions[rule_index][i]

	expr.terms.type in _logical_expr_types

	# `and`/`or` operands count as constant here, as whether they really are is
	# determined by their own operands, which the walk below covers as well
	constant_locations := _constant_expr_locations[rule_index] | _logical_expr_locations[rule_index]

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
}

# METADATA
# description: |
#   true if the `and`/`or` operand is made up of a single constant expression, like
#   the `1` in `input.a and 1`, or the `1 or 2` in `input.a and { 1 or 2 }`
_constant_operand(rule_index, operand) if {
	count(operand) == 1

	operand[0].location in _constant_expr_locations[rule_index]
} else if {
	count(operand) == 1

	operand[0].location in _constant_logical_expr_locations[rule_index]
}

# METADATA
# description: |
#   the location covering both the constant operand on 'side' of the `and`/`or` expression
#   and the operator, i.e. the range that when removed collapses the expression to the
#   operand kept
# scope: document
_collapse_location(expr, "rhs") := $"{keep.end.row}:{keep.end.col}:{whole.end.row}:{whole.end.col}" if {
	keep := util.to_location_no_text(expr.terms.lhs[0].location)
	whole := util.to_location_no_text(expr.location)
}

_collapse_location(expr, "lhs") := $"{whole.row}:{whole.col}:{keep.row}:{keep.col}" if {
	keep := util.to_location_no_text(expr.terms.rhs[0].location)
	whole := util.to_location_no_text(expr.location)
}

# METADATA
# description: |
#   true if the operand on 'side' is a single expression that isn't brace enclosed, and so
#   can replace the `and`/`or` expression it's part of
_stands_alone(terms, side) if {
	count(terms[side]) == 1

	not terms[$"explicit_{side}"]
}
