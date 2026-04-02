# METADATA
# description: Custom function may be replaced by `in` keyword
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-in-construct
package regal.rules.idiomatic["custom-in-construct"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in ast.functions

	# while there could be more convoluted ways of doing this
	# we'll settle for the likely most common case (`item == coll[_]`)
	count(rule.body) == 1

	terms := rule.body[0].terms

	terms[0].value[0].type == "var"
	terms[0].value[0].value in {"eq", "equal"}

	[var, ref] := _normalize_eq_terms(terms)

	arg_names := ast.function_arg_names(rule)

	var.value in arg_names
	ref.value[0].value in arg_names
	ast.is_wildcard(ref.value[1])

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

# normalize var to always always be on the left hand side
_normalize_eq_terms(terms) := [terms[1], terms[2]] if {
	terms[1].type == "var"
	terms[2].type == "ref"
	terms[2].value[0].type == "var"
}

_normalize_eq_terms(terms) := [terms[2], terms[1]] if {
	terms[1].type == "ref"
	terms[1].value[0].type == "var"
	terms[2].type == "var"
}
