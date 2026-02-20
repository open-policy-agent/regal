# METADATA
# description: Use the `contains` keyword
package regal.rules.idiomatic["use-contains"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result
import data.regal.util

# METADATA
# description: Missing capability for keyword `contains`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_contains

# METADATA
# description: Since OPA 1.0, use-contains enabled only when provided a v0 policy
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

report contains violation if {
	# if rego.v1 is imported, OPA will ensure this anyway
	not ast.imports_has_path(ast.imports, ["rego", "v1"])

	some rule in ast.rules

	rule.head.key
	not rule.head.value

	text := util.substring_to(util.to_location_object(rule.location).text, 0, "\n")

	not contains(text, " contains ")

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}
