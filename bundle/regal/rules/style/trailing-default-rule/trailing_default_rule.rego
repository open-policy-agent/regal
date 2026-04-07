# METADATA
# description: Default rule should be declared first
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/trailing-default-rule
package regal.rules.style["trailing-default-rule"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i, rule in input.rules

	rule.default == true

	name := ast.ref_to_string(rule.head.ref)
	name in _all_names(array.slice(input.rules, 0, i))

	violation := result.fail(rego.metadata.chain(), result.location(rule))
}

_all_names(rules) := {name |
	some rule in rules
	name := ast.ref_to_string(rule.head.ref)
}
