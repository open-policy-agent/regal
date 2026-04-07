# METADATA
# description: Use the `if` keyword
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-if
package regal.rules.idiomatic["use-if"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result
import data.regal.util

# METADATA
# description: Missing capability for keyword `if`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_if

# METADATA
# description: Since OPA 1.0, use-if enabled only when provided a v0 policy
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

report contains violation if {
	# if rego.v1 is imported, OPA will ensure this anyway
	not ast.imports_has_path(ast.imports, ["rego", "v1"])

	some rule in input.rules
	rule.body

	head_len := count(util.to_location_object(rule.head.location).text)
	text := trim_space(substring(util.to_location_object(rule.location).text, head_len, -1))

	not startswith(text, "if")

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(rule.head.ref))
}
