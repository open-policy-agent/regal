# METADATA
# description: Avoid chaining rule bodies
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/custom/chained-rule-body
package regal.rules.custom["chained-rule-body"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in input.rules

	ast.is_chained_rule_body(rule, input.regal.file.lines)

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}
