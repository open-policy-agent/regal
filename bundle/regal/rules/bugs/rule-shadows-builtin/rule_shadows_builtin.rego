# METADATA
# description: Rule name shadows built-in
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/rule-shadows-builtin
package regal.rules.bugs["rule-shadows-builtin"]

import data.regal.ast
import data.regal.result

report contains violation if {
	head := input.rules[_].head

	ast.ref_to_string(head.ref) in ast.builtin_namespaces

	violation := result.fail(rego.metadata.chain(), result.location(head))
}
