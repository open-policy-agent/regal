# METADATA
# description: Non-loop expression
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/performance/non-loop-expression
package regal.rules.performance["non-loop-expression"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index, start_points in _loop_start_points

	first_loop_row := min(object.keys(start_points))

	some row, expr
	_exprs[rule_index][row][expr]

	row > first_loop_row

	# users are able to use print statements for debugging purposes
	not _is_print_call(expr.terms[0])

	# if there are any term vars used in the expression, then they must have been
	# declared after the first loop
	every term_var in _expr_vars(expr) {
		not startswith(term_var.value, "$")

		declared_row := min(object.get(_assignment_index, [rule_index, term_var.value], {0}))

		declared_row < first_loop_row
	}

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

# special case for every, as variables declared don't escape its scope
report contains violation if {
	some rule_index
	_every := ast.found.every[rule_index][_]

	kv_vars := {_every[kind].value |
		some kind in ["key", "value"]

		_every[kind].type == "var"
	}

	some expr in _every.body

	not _is_print_call(expr.terms[0])
	not _any_var_found(expr, _every.body, kv_vars)

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

_is_print_call(term) if {
	term.type == "ref"
	term.value[0].value == "print"
}

_any_var_found(expr, _, vars) if {
	some term in _expr_vars(expr)

	term.value in vars
}

_any_var_found(expr, body, _) if {
	assigned := _assign_vars(body)

	some term in _expr_vars(expr)

	term.value in assigned
}

_assign_vars(body) := {expr.terms[1].value |
	some expr in body

	expr.terms[0].type == "ref"
	expr.terms[0].value[0].value == "assign"
}

_expr_vars(expr) := _term_vars(expr.terms) if not expr.with
_expr_vars(expr) := array.flatten([_term_vars(expr.terms), _vars_no_builtins(expr.with)]) if expr.with

_term_vars(terms) := _vars_no_builtins(terms[2]) if {
	terms[0].type == "ref"
	terms[0].value[0].value == "assign"
	terms[0].value[0].type == "var"
} else := _vars_no_builtins(terms)

_vars_no_builtins(terms) := [term |
	some term in ast.find_term_vars(terms)

	not term.value in ast.builtin_names
]

_exprs[rule_index][row] contains expr if {
	some rule_index
	expr := input.rules[rule_index].body[_]
	row := to_number(substring(expr.location, 0, indexof(expr.location, ":")))
}

# covers assigned var from iteration, e.g. x in:
# x := foo.bar[_]
# x = foo.bar[y]
_loop_start_points[rule_index][row] contains var if {
	some rule_index, row
	expr := _exprs[rule_index][row][_]

	expr.terms[0].type == "ref"
	expr.terms[0].value[0].value in {"eq", "assign"}
	expr.terms[1].type == "var"
	expr.terms[2].type == "ref"

	# right hand side is a "loop ref"
	# while a left hand side loop ref is possible, e.g. foo.bar[_] = x,
	# it's both ugly and uncommon enough that we can ignore it for now
	_loop_refs[rule_index][expr.terms[2].location]

	var := expr.terms[1]
	# no need to ignore vars here in comprehensions, since we are only looking
	# for top level wildcards in the final term.
}

# covers output vars in refs, e.g. y in:
# foo.bar[_][y]
# x := foo.bar[y]
_loop_start_points[rule_index][loc.row] contains term if {
	some rule_index, location, term
	_loop_refs[rule_index][location][term]

	not startswith(term.value, "$")

	loc := util.to_location_object(location)

	# ignore vars in comprehensions
	_not_in_comprehension(rule_index, loc)
}

# cover iteration in the form of:
# some x in foo.bar
# some x, y in foo.bar
_loop_start_points[rule_index][loc.row] contains var if {
	some rule_index
	var := ast.found.vars[rule_index].somein[_]

	loc := util.to_location_object(var.location)

	# ignore vars in comprehensions
	_not_in_comprehension(rule_index, loc)
}

_loop_start_points[rule_index][row] contains var if {
	some rule_index, call
	ast.function_calls[rule_index][call].name == "walk"

	call.args[1].type == "array"

	some var in ast.find_term_vars(call.args[1].value)

	row := to_number(substring(var.location, 0, indexof(var.location, ":")))
}

_loop_refs[rule_index][ref.location] contains term if {
	some rule_index, ref, i
	ast.found.refs[rule_index][ref].value[i].type == "var"
	i > 0

	term := ast.found.refs[rule_index][ref].value[i]

	ast.is_output_var(input.rules[rule_index], term)
}

_assignment_index[rule_index][var_value] contains row if {
	some rule_index, row
	var_value := _loop_start_points[rule_index][row][_].value
}

_assignment_index[rule_index][var.value] contains loc.row if {
	some rule_index
	var := ast.found.vars[rule_index].assign[_]
	loc := util.to_location_object(var.location)

	# ignore vars in comprehensions
	_not_in_comprehension(rule_index, loc)
}

_not_in_comprehension(rule_index, loc) if {
	comps := object.get(ast.found.comprehensions, rule_index, set())

	every comp in comps {
		comp_loc := util.to_location_object(comp.location)
		range := [[comp_loc.row, comp_loc.col], [comp_loc.end.row, comp_loc.end.col]]

		not util.point_in_range([loc.row, loc.col], range)
		not util.point_in_range([loc.end.row, loc.end.col], range)
	}
}
