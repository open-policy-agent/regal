# METADATA
# description: Redundant count before loop
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/redundant-loop-count
package regal.rules.bugs["redundant-loop-count"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in input.rules
	some i, expr in rule.body

	# 1st expression, count of $ref compared either to > 0 or != 0
	expr.terms[0].type == "ref"
	expr.terms[0].value[0].type == "var"
	expr.terms[0].value[0].value in {"gt", "neq"}
	expr.terms[1].type == "call"
	expr.terms[1].value[0].type == "ref"
	expr.terms[1].value[0].value[0].value == "count"
	expr.terms[2].value == 0

	# 2nd expression, some x in $ref or some x, y in $ref
	next := rule.body[i + 1]
	next.terms.symbols[0].type == "call"
	next.terms.symbols[0].value[0].type == "ref"
	next.terms.symbols[0].value[0].value[0].value == "internal"
	next.terms.symbols[0].value[0].value[1].value in {"member_2", "member_3"}

	# Last, ensure that the same ref that was counted is the one iterated over
	term := expr.terms[1].value[1]
	term.type == "ref"

	other := regal.last(next.terms.symbols[0].value)
	other.type == "ref"

	count(term.value) == count(other.value)
	ast.is_terms_subset(term.value, other.value)

	violation := result.fail(rego.metadata.chain(), result.location(expr.terms[1]))
}
