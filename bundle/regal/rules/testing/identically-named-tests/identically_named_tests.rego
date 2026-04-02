# METADATA
# description: Multiple tests with same name
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/identically-named-tests
package regal.rules.testing["identically-named-tests"]

import data.regal.ast
import data.regal.result

report contains violation if {
	test_names := [ast.ref_to_string(rule.head.ref) | some rule in ast.tests]

	some i, name in test_names

	name in array.slice(test_names, 0, i)

	violation := result.fail(rego.metadata.chain(), result.location(_rule_by_name(name, ast.tests).head))
}

_rule_by_name(name, rules) := regal.last([rule |
	some rule in rules
	rule.head.ref[0].value == name
])
