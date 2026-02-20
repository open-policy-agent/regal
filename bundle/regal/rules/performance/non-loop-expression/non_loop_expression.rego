# METADATA
# description: Non-loop expression
package regal.rules.performance["non-loop-expression"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index, sps in _loop_start_points
	first_loop_row := min(object.keys(sps))

	some row, expr
	_exprs[rule_index][row][expr]
	row > first_loop_row

	term_vars := ast.find_term_vars(expr)

	# users are able to use print statements for debugging purposes.
	# Continued use is detected by another rule.
	term_vars[0].value != "print"

	# if there are any term vars used in the expression, then they must have been
	# declared after the first loop
	every term_var in term_vars {
		term_var_rows := object.get(_assignment_index, [rule_index, term_var.value], {0})
		min(term_var_rows) < first_loop_row
	}

	violation := result.fail(rego.metadata.chain(), result.location(expr))
}

_exprs[rule_index][row] contains expr if {
	some i
	expr := input.rules[i].body[_]
	rule_index := ast.rule_index_strings[i]
	row := to_number(substring(expr.location, 0, indexof(expr.location, ":")))
}

_exprs[rule_index][row] contains expr if {
	some i
	expr := input.rules[i].body[_].terms.body[_]
	rule_index := ast.rule_index_strings[i]
	row := to_number(substring(expr.location, 0, indexof(expr.location, ":")))
}

# cover iteration in the form of:
# x := foo.bar[_]
# x = foo.bar[_]
_loop_start_points[rule_index][loc.row] contains var if {
	some rule_index
	term := _exprs[rule_index][_][_][_]

	is_array(term)

	last_term := regal.last(term)
	last_term.type == "ref"

	last := regal.last(last_term.value)
	last.type == "var"
	startswith(last.value, "$")

	some var in ast.found.vars[rule_index].assign

	loc := util.to_location_object(var.location)
	loc.row == to_number(util.substring_to(term[0].location, 0, ":"))
	# no need to ignore vars here in comprehensions, since we are only looking
	# for top level wildcards in the final term.
}

# cover iteration in the form of:
# some x in foo.bar
# some x, y in foo.bar
# every x, y in foo.bar
_loop_start_points[rule_index][loc.row] contains var if {
	some rule_index
	some context in ["some", "somein", "every"]
	var := ast.found.vars[rule_index][context][_]

	loc := util.to_location_object(var.location)

	# ignore vars in comprehensions
	comps := object.get(ast.found.comprehensions, rule_index, set())
	every comp in comps {
		comp_loc := util.to_location_object(comp.location)
		range := [[comp_loc.row, comp_loc.col], [comp_loc.end.row, comp_loc.end.col]]
		not util.point_in_range([loc.row, loc.col], range)
		not util.point_in_range([loc.end.row, loc.end.col], range)
	}
}

_loop_start_points[rule_index][row] contains var if {
	some rule_index, call
	ast.function_calls[rule_index][call].name == "walk"

	call.args[1].type == "array"

	some var in ast.find_term_vars(call.args[1].value)
	row := to_number(substring(var.location, 0, indexof(var.location, ":")))
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
	comps := object.get(ast.found.comprehensions, rule_index, set())
	every comp in comps {
		comp_loc := util.to_location_object(comp.location)
		range := [[comp_loc.row, comp_loc.col], [comp_loc.end.row, comp_loc.end.col]]
		not util.point_in_range([loc.row, loc.col], range)
		not util.point_in_range([loc.end.row, loc.end.col], range)
	}
}
