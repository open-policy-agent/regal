# METADATA
# description: Call to `walk` can be optimized
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/performance/walk-no-path
package regal.rules.performance["walk-no-path"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule_index, call
	ast.function_calls[rule_index][call].name == "walk"

	call.args[1].type == "array"
	call.args[1].value[0].type == "var"

	path_var := call.args[1].value[0]

	not startswith(path_var.value, "$")
	not ast.var_in_head(input.rules[rule_index].head, path_var.value)

	not _var_in_other_call(ast.function_calls, rule_index, path_var)
	not _var_value_in_ref(rule_index, path_var.value)

	violation := result.fail(rego.metadata.chain(), result.location(call))
}

# similar to ast.var_in_call, but here we need to discern that
# the var is not the same var declared in the walk itself
_var_in_other_call(calls, rule_index, var) if _var_in_arg(calls[rule_index][_].args[_], var)

_var_in_arg(arg, var) if {
	arg.type == "var"
	arg.value == var.value

	arg.location != var.location
}

_var_in_arg(arg, var) if {
	arg.type in {"array", "object", "set"}

	some term_var in ast.find_term_vars(arg)

	term_var.value == var.value
	term_var.location != var.location
}

_var_value_in_ref(rule_index, value) if ast.found.vars[rule_index].ref[_].value == value
_var_value_in_ref(rule_index, value) if ast.found.refs[rule_index][_].value[0].value == value
