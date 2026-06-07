# METADATA
# description: Repeated computation
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/performance/repeated-computation
package regal.rules.performance["repeated-computation"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index
	call := _calls[rule_index][_]

	some prior in _calls[rule_index]
	call.key == prior.key
	_before(prior.location, call.location)

	violation := result.fail(rego.metadata.chain(), result.ranged_location_between(call.term[0], regal.last(call.term)))
}

_calls[rule_index] contains call_info if {
	some rule_index
	call := ast.found.calls[rule_index][_]

	name := ast.ref_to_string(call[0].value)
	name in ast.builtin_names
	not _excluded_builtin(name)

	args := array.slice(call, 1, 100)
	every arg in args {
		_stable_arg(arg)
	}

	location := util.to_location_object(call[0].location)
	_top_level_body_call(rule_index, location)
	not _in_comprehension(rule_index, location)
	not _in_every_body(rule_index, location)

	call_info := {
		"key": {
			"name": name,
			"args": [_without_locations(arg) | some i, arg in args],
		},
		"location": location,
		"term": call,
	}
}

_stable_arg(arg) if ast.is_constant(arg)

_stable_arg(arg) if {
	arg.type == "ref"
	arg.value[0].type == "var"
	arg.value[0].value in {"data", "input"}
	ast.static_ref(arg)
}

_excluded_builtin(name) if name in ast.operators

_excluded_builtin(name) if name in {"print", "trace"}

_excluded_builtin(name) if object.get(config.capabilities.builtins[name], "nondeterministic", false)

_top_level_body_call(rule_index, location) if {
	expr := input.rules[rule_index].body[_]
	not expr.with

	_contains_location(util.to_location_object(expr.location), location)
}

_in_comprehension(rule_index, location) if {
	comprehension := ast.found.comprehensions[rule_index][_]

	_contains_location(util.to_location_object(comprehension.location), location)
}

_in_every_body(rule_index, location) if {
	_every := ast.found.every[rule_index][_]
	expr := _every.body[_]

	_contains_location(util.to_location_object(expr.location), location)
}

_contains_location(outer, inner) if {
	range := [[outer.row, outer.col], [outer.end.row, outer.end.col]]

	util.point_in_range([inner.row, inner.col], range)
	util.point_in_range([inner.end.row, inner.end.col], range)
}

_before(left, right) if left.row < right.row

_before(left, right) if {
	left.row == right.row
	left.col < right.col
}

_without_locations(value) := [_without_locations(item) | some i, item in value] if is_array(value)

_without_locations(value) := {key: _without_locations(item) |
	is_object(value)
	some key, item in value
	key != "location"
}

_without_locations(value) := {_without_locations(item) | some item in value} if is_set(value)

_without_locations(value) := value if {
	not is_array(value)
	not is_object(value)
	not is_set(value)
}
