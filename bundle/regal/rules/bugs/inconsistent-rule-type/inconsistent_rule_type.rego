# METADATA
# description: Inconsistent rule head value types
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/inconsistent-rule-type
package regal.rules.bugs["inconsistent-rule-type"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i, rule1 in input.rules
	not rule1.head.args
	not rule1.head.key
	count(rule1.head.ref) == 1
	type1 := _head_value_type(rule1)
	type1 != ""

	name := ast.ref_to_string(rule1.head.ref)

	some j, rule2 in input.rules
	i < j
	not rule2.head.args
	not rule2.head.key
	count(rule2.head.ref) == 1
	ast.ref_to_string(rule2.head.ref) == name

	type2 := _head_value_type(rule2)
	type2 != ""
	type1 != type2

	# Flag when combining boolean and non-boolean types (e.g. default false vs string)
	_is_boolean_mismatch(type1, type2)

	violation := result.fail(rego.metadata.chain(), result.location(rule2.head))
}

_is_boolean_mismatch("boolean", t) if t != "boolean"

_is_boolean_mismatch(t, "boolean") if t != "boolean"

_head_value_type(rule) := type if {
	type := rule.head.value.type
	type in (ast.scalar_types | {"array", "object", "set"})
} else := ""
