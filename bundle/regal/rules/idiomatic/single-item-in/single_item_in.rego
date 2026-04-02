# METADATA
# description: Avoid `in` for single item collection
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/single-item-in
package regal.rules.idiomatic["single-item-in"]

import data.regal.ast
import data.regal.result

report contains violation if {
	call := ast.found.calls[_][_]

	call[0].value[0].value == "internal"
	call[0].value[1].value == "member_2"

	call[2].type in {"array", "set", "object"}
	count(call[2].value) == 1

	violation := result.fail(rego.metadata.chain(), result.location(call))
}
