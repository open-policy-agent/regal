# METADATA
# description: Import shadows rule
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/import-shadows-rule
package regal.rules.bugs["import-shadows-rule"]

import data.regal.ast
import data.regal.result

report contains violation if {
	head := input.rules[_].head

	count(head.ref) == 1
	head.ref[0].value in ast.imported_identifiers

	violation := result.fail(rego.metadata.chain(), result.location(head))
}
