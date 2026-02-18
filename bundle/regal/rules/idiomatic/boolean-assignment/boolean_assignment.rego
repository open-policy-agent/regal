# METADATA
# description: Prefer `if` over boolean assignment
package regal.rules.idiomatic["boolean-assignment"]

import data.regal.config
import data.regal.result

report contains violation if {
	head := input.rules[_].head
	rhv := head.value

	rhv.type == "call"
	rhv.value[0].type == "ref"

	ref_name := rhv.value[0].value[0].value

	config.capabilities.builtins[ref_name].decl.result == "boolean"

	violation := result.fail(rego.metadata.chain(), result.location(head))
}
