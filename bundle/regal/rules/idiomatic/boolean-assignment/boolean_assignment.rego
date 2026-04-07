# METADATA
# description: Prefer `if` over boolean assignment
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/boolean-assignment
package regal.rules.idiomatic["boolean-assignment"]

import data.regal.config
import data.regal.result

report contains violation if {
	head := input.rules[_].head
	term := head.value

	term.type == "call"
	term.value[0].type == "ref"

	ref_name := term.value[0].value[0].value

	config.capabilities.builtins[ref_name].decl.result == "boolean"

	violation := result.fail(rego.metadata.chain(), result.location(head))
}
