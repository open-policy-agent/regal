# METADATA
# description: Redundant existence check
package regal.rules.bugs["redundant-existence-check"]

import data.regal.ast
import data.regal.result

# METADATA
# description: check rule bodies for redundant existence checks
report contains violation if {
	some rule_index, rule in input.rules
	some expr_index, expr in _exprs[rule_index]

	expr.terms.type == "ref"

	not expr.with

	not rule.body[expr_index + 1].negated
	ast.static_ref(expr.terms)

	some term in rule.body[expr_index + 1].terms

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
	some expr in func.body

	not expr.negated
	expr.terms.type == "var"

	some arg in func.head.args

	arg.type == "var"
	arg.value == expr.terms.value

	violation := result.fail(rego.metadata.chain(), result.location(expr.terms))
}

# METADATA
# description: check for redundant existence checks in rule head assignment
report contains violation if {
	some rule_index, rule in input.rules

	rule.head.value.type == "ref"

	some expr in _exprs[rule_index]

	not expr.negated
	expr.terms.type == "ref"
	ast.is_terms_subset(expr.terms.value, rule.head.value.value)

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(expr.terms.value))
}

# all top-level expressions in module
_exprs[rule_index][expr_index] := expr if {
	some rule_index, rule in input.rules
	some expr_index, expr in rule.body
}
