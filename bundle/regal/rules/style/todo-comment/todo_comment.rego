# METADATA
# description: Avoid TODO comments
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/todo-comment
package regal.rules.style["todo-comment"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some location in ast.comments_decoded

	regex.match(`(?i)^#\s*(todo|fixme)`, location.text)

	violation := result.fail(rego.metadata.chain(), result.with_text(location))
}
