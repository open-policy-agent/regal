# METADATA
# description: Naming convention violation
package regal.rules.custom["naming-convention"]

import data.regal.ast
import data.regal.config
import data.regal.result

# target: package
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	"package" in convention.targets

	not regex.match(convention.pattern, ast.package_name)

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(input.package, _message("package", ast.package_name, convention.pattern)),
	)
}

# target: rule
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	"rule" in convention.targets

	some rule in ast.rules

	name := ast.ref_to_string(rule.head.ref)

	not regex.match(convention.pattern, name)

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(rule.head, _message("rule", name, convention.pattern)),
	)
}

# target: function
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	"function" in convention.targets

	some rule in ast.functions

	name := ast.ref_to_string(rule.head.ref)

	not regex.match(convention.pattern, name)

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(rule.head, _message("function", name, convention.pattern)),
	)
}

# target: var
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	some target in convention.targets

	target in {"var", "variable"}

	var := ast.found.vars[_][_][_]

	not regex.match(convention.pattern, var.value)

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(var, _message("variable", var.value, convention.pattern)),
	)
}

_message(kind, name, pattern) := $`Naming convention violation: {kind} name "{name}" does not match pattern '{pattern}'`
