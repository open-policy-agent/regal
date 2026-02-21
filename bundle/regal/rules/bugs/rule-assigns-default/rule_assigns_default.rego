# regal eval:use-as-input
# METADATA
# description: Rule assigned its default value
package regal.rules.bugs["rule-assigns-default"]

import data.regal.ast
import data.regal.result

report contains violation if {
	_default_rule_values != {}

	some i, rule in input.rules

	not rule.default

	_default_rule_values[ast.rule_names_ordered[i]] == rule.head.value.value

	violation := result.fail(rego.metadata.chain(), result.location(rule.head.value))
}

_default_rule_values[ref] := rule.head.value.value if {
	some rule in input.rules
	rule.default

	ref := ast.ref_to_string(rule.head.ref)
}
