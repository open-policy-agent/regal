# METADATA
# description: Prefer to use `object.keys`
package regal.rules.idiomatic["use-object-keys"]

import data.regal.ast
import data.regal.result

# {k | some k, _ in object}
report contains violation if {
	comprehension := ast.found.comprehensions[_][_]

	comprehension.type == "setcomprehension"
	comprehension.value.term.type == "var"
	count(comprehension.value.body) == 1

	symbol := comprehension.value.body[0].terms.symbols[0]

	# some k, _ in object
	symbol.type == "call"
	symbol.value[0].type == "ref"
	symbol.value[0].value[0].value == "internal"
	symbol.value[0].value[1].value == "member_3"

	# same 'k' var value as in the head
	symbol.value[1].type == "var"
	symbol.value[1].value == comprehension.value.term.value

	# we don't care what symbol.value[2] is, as it's not assigned in the head
	# but symbol.value[3] should be a ref without vars

	symbol.value[3].type in {"ref", "var"}

	ast.static_ref(symbol.value[3].value)

	violation := result.fail(rego.metadata.chain(), result.location(comprehension))
}

# {k | input.object[k]}
# {k | some k; input.object[k]}
report contains violation if {
	comprehension := ast.found.comprehensions[_][_]

	comprehension.type == "setcomprehension"
	comprehension.value.term.type == "var"

	ref := _ref(comprehension.value.body)

	vars := [term |
		some term in array.slice(ref, 1, 100)
		term.type == "var"
	]

	count(vars) == 1
	vars[0].value == comprehension.value.term.value

	violation := result.fail(rego.metadata.chain(), result.location(comprehension))
}

# {k | input.object[k]}
_ref(exprs) := exprs[0].terms.value if {
	count(exprs) == 1
	exprs[0].terms.type == "ref"
}

# {k | some k; input.object[k]}
_ref(exprs) := exprs[1].terms.value if {
	count(exprs) == 2
	exprs[0].terms.symbols[0].type == "var"
	exprs[1].terms.type == "ref"
}
