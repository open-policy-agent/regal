# METADATA
# description: Superfluous `object.get` call
package regal.rules.idiomatic["superfluous-object-get"]

import data.regal.ast
import data.regal.result

# object.get(a, "b", "c") == "violation"
report contains violation if {
	# check the most unlikely case first in order to fail fast
	some rule_index, expr
	ast.found.expressions[rule_index][expr].terms[1].type == "call"

	terms := ast.found.expressions[rule_index][expr].terms

	terms[1].value[0].value[0].value == "object"
	terms[1].value[0].value[1].value == "get"

	terms[0].type == "ref"
	terms[0].value[0].type == "var"
	terms[0].value[0].value == "equal"

	def_value := terms[1].value[3]
	rhs_value := terms[2]

	[def_value.type, def_value.value] != [rhs_value.type, rhs_value.value]

	def_value.type != "var"
	def_value.type != "ref"

	violation := result.fail(rego.metadata.chain(), result.location(terms[1]))
}

# "violation" == object.get(a, "b", "c")
report contains violation if {
	# check the most unlikely case first in order to fail fast
	some rule_index, expr
	ast.found.expressions[rule_index][expr].terms[2].type == "call"

	terms := ast.found.expressions[rule_index][expr].terms

	terms[2].value[0].value[0].value == "object"
	terms[2].value[0].value[1].value == "get"

	terms[0].type == "ref"
	terms[0].value[0].type == "var"
	terms[0].value[0].value == "equal"

	def_value := terms[2].value[3]
	lhs_value := terms[1]

	[def_value.type, def_value.value] != [lhs_value.type, lhs_value.value]

	def_value.type != "var"
	def_value.type != "ref"

	violation := result.fail(rego.metadata.chain(), result.location(terms[2]))
}
