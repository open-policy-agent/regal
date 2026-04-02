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

	block[0].location.col == 1
	ast.comments.annotation_match(block[0].text)

	violation := result.fail(rego.metadata.chain(), result.location(block[0]))
}
