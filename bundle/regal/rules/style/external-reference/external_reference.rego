# METADATA
# description: External reference in function
package regal.rules.style["external-reference"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	some i
	arg_vars := _args_vars(input.rules[i].head.args)
	own_vars := {value | value := ast.found.vars[ast.rule_index_strings[i]][_][_].value}

	# note: parens added by opa fmt 🤦
	allowed_refs := (arg_vars | own_vars) | ast.all_function_namespaces

	external := [value |
		some node in ["head", "body", "else"]

		walk(input.rules[i][node], [path, value])

		value.type == "var"
		not value.value in allowed_refs
		not startswith(value.value, "$")
		not _function_call_ctx(input.rules[i], array.flatten([node, path]))
	]

	count(external) > object.get(config.rules, ["style", "external-reference", "max-allowed"], 2)

	some value in external

	violation := result.fail(rego.metadata.chain(), result.location(value))
}

_args_vars(args) := {name |
	some arg in args
	some name in _named_vars(arg)
}

_named_vars(arg) := {arg.value} if arg.type == "var"
_named_vars(arg) := {var.value | some var in ast.find_term_vars(arg)} if arg.type in {"array", "object", "set"}

# METADATA
# scope: document
# description: |
#   functions should be able to call other functions and this shouldn't
#   be flagged as there's no way to import other functions via arguments
#   note: this doesn't check for built-in calls or calls to function
#   defined in the same package, as those are already covered by
#   "fn_namespaces" in the report rule
_function_call_ctx(fn, path) if {
	object.get(fn, array.slice(path, 0, count(path) - 4), false).type == "call"
} else if {
	terms_path := array.slice(path, 0, util.last_indexof(path, "terms") + 2)
	next_term_path := array.flatten([
		array.slice(terms_path, 0, count(terms_path) - 1), # ["body", 0, "terms", 0] -> ["body", 0, "terms"]
		regal.last(terms_path) + 1, # 0 -> 1
	])

	# ["body", 0, "terms", 1]

	object.get(fn, next_term_path, null) != null
}
