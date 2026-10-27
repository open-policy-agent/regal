# METADATA
# description: Remove future keyword imports that are redundant in Rego v1
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/redundant-future-keywords
package regal.rules.imports["redundant-future-keywords"]

import data.regal.capabilities
import data.regal.result

# METADATA
# description: Rule only applies to Rego v1 capabilities
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if not capabilities.is_opa_v1

report contains violation if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"

	some imported in input.imports
	terms := imported.path.value
	count(terms) == 3
	terms[0].value == "future"
	terms[1].value == "keywords"
	not terms[2].value in {"and", "or", "not"}

	violation := result.fail(rego.metadata.chain(), result.location(imported.path))
}
