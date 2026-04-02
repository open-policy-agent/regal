# METADATA
# description: Messy incremental rule
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/messy-rule
package regal.rules.style["messy-rule"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i, rule in input.rules

	# tests aren't really incremental rules, and other rules
	# will flag multiple rules with the same name
	not startswith(trim_prefix(rule.head.ref[0].value, "todo_"), "test_")

	cur_name := ast.rule_names_ordered[i]

	some j, other in input.rules

	j > i

	nxt_name := ast.rule_names_ordered[j]
	cur_name == nxt_name

	previous_name := ast.rule_names_ordered[j - 1]
	previous_name != nxt_name

	violation := result.fail(rego.metadata.chain(), result.location(other))
}
