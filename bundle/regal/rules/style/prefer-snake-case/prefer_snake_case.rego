# METADATA
# description: Prefer snake_case for names
package regal.rules.style["prefer-snake-case"]

import data.regal.ast
import data.regal.result

report contains violation if {
	ast.package_name != lower(ast.package_name)

	some term in input.package.path

	term.value != lower(term.value)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}

report contains violation if {
	term := input.rules[_].head.ref[_]

	term.value != lower(term.value)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}

report contains violation if {
	var := ast.found.vars[_][_][_]

	var.value != lower(var.value)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}
