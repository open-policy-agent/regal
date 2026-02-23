# METADATA
# description: Prefer snake_case for names
package regal.rules.style["prefer-snake-case"]

import data.regal.ast
import data.regal.result

report contains violation if {
	ast.package_name != lower(ast.package_name)

	some part in input.package.path

	part.value != lower(part.value)

	violation := result.fail(rego.metadata.chain(), result.location(part))
}

report contains violation if {
	part := input.rules[_].head.ref[_]

	part.value != lower(part.value)

	violation := result.fail(rego.metadata.chain(), result.location(part))
}

report contains violation if {
	var := ast.found.vars[_][_][_]

	var.value != lower(var.value)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}
