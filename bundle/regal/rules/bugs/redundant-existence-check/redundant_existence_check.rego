# METADATA
# description: Redundant existence check
package regal.rules.bugs["redundant-existence-check"]

import data.regal.ast
import data.regal.result

# METADATA
# description: check rule bodies for redundant existence checks
report contains violation if {
	some rule_index, expr_index
	expr := _static_ref_exprs[rule_index][expr_index]

	some adjacent in [-1, 1]
	some term in _exprs[rule_index][expr_index + adjacent].terms

	term.type == "ref"
	ast.is_terms_subset(expr.terms.value, term.value)

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(expr.terms.value))
}

# METADATA
# description: |
#  check for redundant existence checks of function args in function bodies
#  note: this only scans "top level" expressions in the function body, and not
#  e.g. those nested inside of comprehensions, every bodies, etc.. while this
#  would certainly be possible, the cost does not justify the benefit, as it's
#  quite unlikely that existence checks are found there
report contains violation if {
	some func in ast.functions

	arg_vars := {term.value |
		some term in func.head.args
		term.type == "var"
	}

	some expr in func.body

	not expr.negated
	expr.terms.type == "var"
	expr.terms.value in arg_vars

	violation := result.fail(rego.metadata.chain(), result.location(expr.terms))
}

# METADATA
# description: check for redundant existence checks in rule head assignment
report contains violation if {
	some rule_index
	input.rules[rule_index].head.value.type == "ref"

	head := input.rules[rule_index].head

	some expr in _exprs[rule_index]

	expr.terms.type == "ref"
	ast.is_terms_subset(expr.terms.value, head.value.value)

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(expr.terms.value))
}

# all top-level expressions in module
_exprs[rule_index][expr_index] := expr if {
	some rule_index, expr_index
	input.rules[rule_index].body[expr_index]

	expr := input.rules[rule_index].body[expr_index]
	not expr.with
	not expr.negated
}

_static_ref_exprs[rule_index][expr_index] := expr if {
	some rule_index, expr_index
	_exprs[rule_index][expr_index].terms.type == "ref"

	expr := _exprs[rule_index][expr_index]
	ast.static_ref(expr.terms)
}
