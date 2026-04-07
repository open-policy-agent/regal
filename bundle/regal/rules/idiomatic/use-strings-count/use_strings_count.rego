# METADATA
# description: Use `strings.count` where possible
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-strings-count
package regal.rules.idiomatic["use-strings-count"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result

# METADATA
# description: Missing capability for built-in function `strings.count`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_strings_count

# METADATA
# description: flag calls to `count` where the first argument is a call to `indexof_n`
report contains violation if {
	"indexof_n" in ast.builtin_functions_called

	ref := ast.found.calls[_][_]

	ref[0].value[0].type == "var"
	ref[0].value[0].value == "count"

	ref[1].type == "call"
	ref[1].value[0].value[0].type == "var"
	ref[1].value[0].value[0].value == "indexof_n"

	violation := result.fail(
		rego.metadata.chain(),
		result.ranged_location_between(
			result.location(ref[0]),
			result.location(ref[1]),
		),
	)
}
