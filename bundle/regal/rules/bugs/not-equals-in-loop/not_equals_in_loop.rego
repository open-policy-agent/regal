# METADATA
# description: Use of != in loop
package regal.rules.bugs["not-equals-in-loop"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule_index, i
	ast.found.expressions[rule_index][i].terms[0].type == "ref"

	expr := ast.found.expressions[rule_index][i]
	not expr.interpolated

	terms := expr.terms

	terms[0].type == "ref"
	terms[0].value[0].type == "var"
	terms[0].value[0].value == "neq"

	some neq_term in array.slice(terms, 1, 100)
	neq_term.type == "ref"

	some value in neq_term.value
	ast.is_wildcard(value)

	violation := result.fail(rego.metadata.chain(), result.location(terms[0]))
}
