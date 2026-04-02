# METADATA
# description: Use explicit future keyword imports
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/implicit-future-keywords
package regal.rules.imports["implicit-future-keywords"]

import data.regal.config
import data.regal.result

# METADATA
# description: Rule made obsolete by rego.v1 capability
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if "rego_v1_import" in config.capabilities.features

report contains violation if {
	some imported in input.imports

	imported.path.type == "ref"

	count(imported.path.value) == 2

	imported.path.value[0].type == "var"
	imported.path.value[0].value == "future"
	imported.path.value[1].type == "string"
	imported.path.value[1].value == "keywords"

	violation := result.fail(rego.metadata.chain(), result.location(imported.path.value[0]))
}
