# METADATA
# description: Inconsistently named function arguments
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/inconsistent-args
package regal.rules.bugs["inconsistent-args"]

import data.regal.ast
import data.regal.result

report contains violation if {
	ast.functions != []

	# comprehension indexing
	function_args_by_name := {name: args_list |
		some i
		name := ast.ref_to_string(ast.functions[i].head.ref)
		args_list := [args |
			some j
			ast.ref_to_string(ast.functions[j].head.ref) == name
			args := ast.functions[j].head.args
		]
		count(args_list) > 1
	}

	some name, args_list in function_args_by_name
	not _arity_mismatch(args_list) # leave that to the compiler

	by_position := [partitioned | # "partition" the args by their position
		some i, _ in args_list[0]
		partitioned := [item[i] | some item in args_list]
	]

	some position in by_position

	_inconsistent_args(position)

	args := _find_function_by_name(name).head.args

	violation := result.fail(rego.metadata.chain(), result.ranged_location_between(args[0], regal.last(args)))
}

_arity_mismatch(args_list) if {
	len := count(args_list[0])
	some arr in args_list
	count(arr) != len
}

_inconsistent_args(position) if {
	named_vars := {arg.value |
		some arg in position
		arg.type == "var"
		not startswith(arg.value, "$")
	}
	count(named_vars) > 1
}

# Return the _second_ function found by name, as that
# is reasonably the location the inconsistency is found
_find_function_by_name(name) := [fun |
	some fun in ast.functions
	ast.ref_to_string(fun.head.ref) == name
][1]
