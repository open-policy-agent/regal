# METADATA
# description: Argument is always a wildcard
package regal.rules.bugs["argument-always-wildcard"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	some name, functions in _function_groups

	fn := util.any_set_item(functions)

	some pos, _ in fn.head.args

	every function in functions {
		function.head.args[pos].type == "var"
		ast.is_wildcard(function.head.args[pos])
	}

	not _function_name_excepted(name)

	violation := result.fail(rego.metadata.chain(), result.location(fn.head.args[pos]))
}

_function_groups[name] contains fn if {
	some fn in ast.functions

	name := ast.ref_to_string(fn.head.ref)
}

_function_name_excepted(name) if {
	regex.match(config.rules.bugs["argument-always-wildcard"]["except-function-name-pattern"], name)
}
