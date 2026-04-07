# METADATA
# description: Prefer default assignment over fallback else
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/default-over-else
package regal.rules.style["default-over-else"]

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	some rule in _considered_rules

	# walking is expensive but necessary here, since there could be
	# any number of `else` clauses nested below. no need to traverse
	# the rule if there isn't a single `else` present though!
	walk(rule.else, [_, value])

	not value.body

	else_head := value.head

	ast.is_constant(else_head.value)

	violation := result.fail(rego.metadata.chain(), result.location(else_head))
}

_considered_rules := input.rules if {
	config.rules.style["default-over-else"]["prefer-default-functions"] == true
} else := ast.rules
