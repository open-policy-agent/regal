# METADATA
# description: Redundant alias
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/redundant-alias
package regal.rules.imports["redundant-alias"]

import data.regal.result

report contains violation if {
	some imported in input.imports

	regal.last(imported.path.value).value == imported.alias

	violation := result.fail(rego.metadata.chain(), result.location(imported.path.value[0]))
}
