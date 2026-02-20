# METADATA
# description: Prefer pattern matching in function arguments
package regal.rules.idiomatic["equals-pattern-matching"]

import data.regal.ast
import data.regal.result

# Current limitations:
# Only works for single comparison either in head or in body

# f(x) := x == 1
# ->
# f(1)
report contains violation if {
	some fn in ast.functions

	not fn.body
	not fn.else

	val := fn.head.value
	val.type == "call"
	val.value[0].type == "ref"
	val.value[0].value[0].value == "equal"

	term := _normalize_eq_terms(val.value, ast.scalar_types)

	some arg in fn.head.args

	arg.type == "var"
	term.value == arg.value

	violation := result.fail(rego.metadata.chain(), result.location(fn))
}

# f(x) if x == 1
# ->
# f(1)
report contains violation if {
	some fn in ast.functions

	fn.body
	not fn.else

	# FOR NOW: Limit to a lone comparison
	# More elaborate cases are certainly doable,
	# but we'd need to keep track of whatever else
	# each var is up to in the body, and that's..
	# well, elaborate.
	count(fn.body) == 1

	expr := fn.body[0]

	expr.terms[0].type == "ref"
	expr.terms[0].value[0].type == "var"
	expr.terms[0].value[0].value == "equal"

	term := _normalize_eq_terms(expr.terms, ast.scalar_types)

	some arg in fn.head.args

	arg.type == "var"
	term.value == arg.value

	violation := result.fail(rego.metadata.chain(), result.location(fn))
}

# normalize var to always always be on the left hand side
_normalize_eq_terms(terms, scalar_types) := terms[1] if {
	not ast.is_wildcard(terms[1])
	terms[2].type in scalar_types
}

_normalize_eq_terms(terms, scalar_types) := terms[2] if {
	terms[1].type in scalar_types
	not ast.is_wildcard(terms[2])
}
