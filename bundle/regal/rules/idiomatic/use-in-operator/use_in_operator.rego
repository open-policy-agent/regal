# METADATA
# description: Use in to check for membership
package regal.rules.idiomatic["use-in-operator"]

import data.regal.ast
import data.regal.result

report contains violation if {
	terms := input.rules[_].body[_].terms

	terms[0].type == "ref"
	terms[0].value[0].value in {"eq", "equal"}

	nl_terms := _non_loop_term(terms)
	count(nl_terms) == 1

	nlt := nl_terms[0]
	_static_term(nlt.term)

	# Use the non-loop term position to determine the
	# location of the loop term (3 is the count of terms)
	violation := result.fail(rego.metadata.chain(), result.location(terms[3 - nlt.pos]))
}

_non_loop_term(terms) := [{"pos": i + 1, "term": term} |
	some i, term in array.slice(terms, 1, 3)
	not _loop_term(term)
]

_loop_term(term) if {
	term.type == "ref"
	term.value[0].type == "var"

	ast.is_wildcard(regal.last(term.value))
}

_static_term(term) if term.type in {"array", "boolean", "object", "null", "number", "set", "string", "var"}

_static_term(term) if {
	term.type == "ref"
	ast.static_ref(term)
}
