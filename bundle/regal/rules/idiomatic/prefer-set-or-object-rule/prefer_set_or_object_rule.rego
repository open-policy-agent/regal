# METADATA
# description: Prefer set or object rule over comprehension
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-set-or-object-rule
package regal.rules.idiomatic["prefer-set-or-object-rule"]

import future.keywords.not
import future.keywords.or

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in ast.rules

	rule.head.value.type in {"setcomprehension", "objectcomprehension"}
	not rule.body

	# Ignore simple conversions from array to set
	not _is_array_conversion(rule.head.value)

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

# {s | some s in arr}
# or
# {s | s := arr[_]}
# or
# {s | s := arr[_].foo}
# or
# {s | s := arr[_].foo[_]}
_is_array_conversion(value) if {
	value.type == "setcomprehension"
	value.value.term.type == "var"

	body := value.value.body

	count(body) == 1
	{
		count(body[0].terms.symbols) == 1
		symbol := body[0].terms.symbols[0]
		symbol.type == "call"
		symbol.value[0].value[0].value == "internal"
		symbol.value[0].value[1].value == "member_2"
		symbol.value[1].value == value.value.term.value
		symbol.value[1].type == "var"
	} or {
		[lhs, rhs] := ast.assignment_terms(body[0].terms)

		# Assignment to comprehension variable
		lhs.value == value.value.term.value
		lhs.type == "var"

		# On the right hand side a ref with at least one wildcard
		rhs.type == "ref"
		rhs.value[0].type == "var"

		some ref_val in rhs.value

		ast.is_wildcard(ref_val)
	}
}
