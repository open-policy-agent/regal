# METADATA
# description: Prefer default assignment over negated condition
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/default-over-not
package regal.rules.style["default-over-not"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i, rule in ast.rules

	# part 1 — find unconditional static ref assignment
	# example: `rule := input.foo`

	not rule.default
	not rule.body

	ast.static_ref(rule.head.value)

	name := ast.ref_static_to_string(rule.head.ref)

	# part 2 - find corresponding assignment of constant on negated condition
	# example: `rule := 1 if not input.foo`

	sibling_rules := [sibling |
		some j, sibling in ast.rules
		i != j
		ast.ref_static_to_string(sibling.head.ref) == name
	]

	some sibling in sibling_rules

	ast.is_constant(sibling.head.value)
	count(sibling.body) == 1
	sibling.body[0].negated

	ast.ref_value_equal(sibling.body[0].terms.value, rule.head.value.value)

	violation := result.fail(rego.metadata.chain(), result.location(sibling.body[0]))
}
