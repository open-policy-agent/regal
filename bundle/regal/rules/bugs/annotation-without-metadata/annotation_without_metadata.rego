# METADATA
# description: Annotation without metadata
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/annotation-without-metadata
package regal.rules.bugs["annotation-without-metadata"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some block in ast.comments.blocks

	block[0].col == 1
	ast.comments.annotation_match(block[0].text)

	violation := result.fail(rego.metadata.chain(), result.with_text(block[0]))
}
