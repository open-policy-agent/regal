# METADATA
# description: Avoid using deprecated built-in functions
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/deprecated-builtin
package regal.rules.bugs["deprecated-builtin"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# description: |
#   Since OPA 1.0, deprecated-builtin enabled only when provided a v0 policy,
#   BUT please note that this may change in the future if new built-in functions
#   are deprecated.
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

report contains violation if {
	deprecated_builtins := {
		"any", "all", "re_match", "net.cidr_overlap", "set_diff", "cast_array",
		"cast_set", "cast_string", "cast_boolean", "cast_null", "cast_object",
	}

	# bail out early if no the deprecated built-ins are in capabilities
	util.intersects(object.keys(config.capabilities.builtins), deprecated_builtins)

	call := ast.found.calls[_][_][0]

	ast.ref_to_string(call.value) in deprecated_builtins

	violation := result.fail(rego.metadata.chain(), result.location(call))
}
