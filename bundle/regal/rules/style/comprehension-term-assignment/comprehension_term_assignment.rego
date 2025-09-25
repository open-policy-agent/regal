# METADATA
# description: Assigned value can be moved to comprehension term
package regal.rules.style["comprehension-term-assignment"]

import data.regal.ast
import data.regal.result

# METADATA
# description: |
#   find comprehensions where assignment in the body may be replaced by
#   using the value in the comprehension term (or key/value) directly
# scope: document

# METADATA
# description: find in set and array comprehensions
report contains violation if {
	comp := ast.found.comprehensions[_][_].value

	# a single expression can't be moved to term position
	count(comp.body) > 1

	# limit to simple vars, not term vars in nested
	# composite structures, function calls or whatever
	comp.term.type == "var"

	some expr in comp.body

	[lhs, rhs] := ast.assignment_terms(expr.terms)

	lhs.type == comp.term.type
	lhs.value == comp.term.value

	# using any of these at the term position may be OK when the value
	# is simple, like [{"first_name": name.first} | some name in names]
	# but almost certainly hard to understand when more complex composite values
	# or call chains are involved..
	# trying to determine "complexity" is... hard / undesirable
	# so let's just allow these assignnments and focus on what we do know
	rhs.type in {"var", "ref"}
	not _dynamic_ref(rhs)

	violation := result.fail(rego.metadata.chain(), result.infix_expr_location(expr.terms))
}

# METADATA
# description: find in object comprehensions (both keys and values)
report contains violation if {
	comp := ast.found.comprehensions[_][_].value

	# a single expression can't be moved to term position
	count(comp.body) > 1

	# only true for object comprehension
	comp.key

	some expr in comp.body

	[lhs, rhs] := ast.assignment_terms(expr.terms)

	some kind in ["key", "value"]

	lhs.type == comp[kind].type
	lhs.value == comp[kind].value

	rhs.type in {"var", "ref"}
	not _dynamic_ref(rhs)

	violation := result.fail(rego.metadata.chain(), result.infix_expr_location(expr.terms))
}

_dynamic_ref(value) if {
	value.type == "ref"

	_call_or_non_static(value)
}

_call_or_non_static(ref) if ref.value[0].type == "call"
_call_or_non_static(ref) if not ast.static_ref(ref)
