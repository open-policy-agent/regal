# METADATA
# description: TODO test encountered
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/todo-test
package regal.rules.testing["todo-test"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in ast.rules

	startswith(ast.ref_to_string(rule.head.ref), "todo_test_")

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}
