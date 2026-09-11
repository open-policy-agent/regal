# METADATA
# description: Naming convention violation
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/custom/naming-convention
package regal.rules.custom["naming-convention"]

import future.keywords.not
import future.keywords.or

import data.regal.ast
import data.regal.config
import data.regal.result

# target: package
report contains violation if {
	_any_package_convention_violation

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(input.package, _message("package", ast.package_name)),
	)
}

# target: rule
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	"rule" in convention.targets

	some rule in ast.rules

	name := ast.ref_to_string(rule.head.ref)
	not {
		name in convention.names or regex.match(convention.pattern, name)
	}

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(rule.head, _message("rule", name)),
	)
}

# target: function
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	"function" in convention.targets

	some rule in ast.functions

	name := ast.ref_to_string(rule.head.ref)
	not {
		name in convention.names or regex.match(convention.pattern, name)
	}

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(rule.head, _message("function", name)),
	)
}

# target: var
report contains violation if {
	some convention in config.rules.custom["naming-convention"].conventions
	some target in convention.targets

	target in {"var", "variable"}

	var := ast.found.vars[_][_][_]

	not startswith(var.value, "$")
	not {
		var.value in convention.names or regex.match(convention.pattern, var.value)
	}

	violation := result.fail(
		rego.metadata.chain(),
		result.location_and_description(var, _message("variable", var.value)),
	)
}

_any_package_convention_violation if {
	some convention in config.rules.custom["naming-convention"].conventions

	"package" in convention.targets
	not {
		ast.package_name in convention.names or regex.match(convention.pattern, ast.package_name)
	}
}

_message(kind, name) := $`Naming violation: {kind} name "{name}" does not match configured convention`
