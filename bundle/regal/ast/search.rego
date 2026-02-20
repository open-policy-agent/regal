package regal.ast

import data.regal.util

_find_nested_vars(obj) := [value |
	walk(obj, [_, value])
	value.type == "var"
	indexof(value.value, "$") == -1
]

# simple assignment, i.e. `x := 100` returns `x`
# always returns a single var, but wrapped in an
# array for consistency or
# 'destructuring' array assignment, i.e.
# [a, b, c] := [1, 2, 3] or {a: b} := {"foo": "bar"}
_find_assign_vars(value) := [value] if {
	value.type == "var"
} else := _find_nested_vars(value) if {
	value.type in {"array", "object"}
}

# METADATA
# description: |
#   true if a var of 'name' can be found in the provided AST node
has_named_var(node, name) if {
	node.type == "var"
	node.value == name
} else if {
	node.type in {"array", "object", "set", "ref", "templatestring"}

	walk(node.value, [_, nested])

	nested.type == "var"
	nested.value == name
}

# METADATA
# description: |
#   traverses all nodes in provided terms (using `walk`), and returns an array with
#   all variables declared in terms, i,e [x, y] or {x: y}, etc.
find_term_vars(terms) := [term |
	walk(terms, [_, term])

	term.type == "var"
]

# METADATA
# description: |
#   traverses all nodes in provided terms (using `walk`), and returns true if any variable
#   is found in terms, with early exit (as opposed to find_term_vars)
has_term_var(terms) if {
	walk(terms, [_, term])

	term.type == "var"
}

# converting to string until https://github.com/open-policy-agent/opa/issues/6736 is fixed
_rule_index(rule) := rule_index_strings[i] if {
	some i, r in _rules
	r == rule
}

# hack to work around the different input models of linting vs. the lsp package.. we
# should probably consider something more robust
_rules := input.rules
_rules := data.workspace.parsed[input.regal.file.uri].rules if not input.rules

# even worse hack to support the new LSP router, and to allow those handlers to use code from
# the AST package. object.get used here to circument schema validation of input..
# there's no doubt that we need to find a better model for this going forward
_rules := data.workspace.parsed[object.get(input, ["params", "textDocument", "uri"], null)].rules if not input.rules

# METADATA:
# description: |
#   object containing all variables found in the input AST, keyed first by the index of
#   the rule where the variables were found (as a numeric string), and then the context
#   of the variable, which will be one of:
#   - args
#   - term
#   - assign
#   - every
#   - some
#   - somein
#   - ref
# scope: document
found.vars[rule_index].every contains term if {
	some rule_index, i
	found.expressions[rule_index][i].terms.domain

	some kind in ["key", "value"]
	term := found.expressions[rule_index][i].terms[kind]

	term.type == "var"
	indexof(term.value, "$") == -1
}

found.vars[rule_index].args contains term if {
	some i, rule_index in rule_index_strings
	some term in _rules[i].head.args

	term.type == "var"
}

found.vars[rule_index].args contains node if {
	some i, rule_index in rule_index_strings
	some term in _rules[i].head.args

	term.type == "array" # only composite type that can contain vars in args position (right?)

	some item in term.value
	not item.type in {"string", "number", "boolean", "null"}

	walk(item, [_, node])

	node.type == "var"
}

found.vars[rule_index].ref contains term if {
	some rule_index, ref
	found.refs[rule_index][ref]

	some x, term in ref.value

	x > 0
	term.type == "var"
}

# find "return value vars", i.e. those placed after a functions declared
# number of args, e.g `x` in `count(x, y)`
found.vars[rule_index].term contains var if {
	some rule_index, calls in found.calls
	some call in calls

	call[0].value[0].type == "var"
	call[0].value[0].value != "assign"

	fn_name := ref_static_to_string(call[0].value)
	undeclared_start := count(all_functions[fn_name].decl.args) + 1

	call[undeclared_start]
	fn_name != "print"

	some var in find_term_vars(array.slice(call, undeclared_start, 100))
}

# `=` isn't necessarily assignment, and only considering the variable on the
# left-hand side is equally dubious, but we'll treat `x = 1` as `x := 1` for
# the purpose of this function until we have a more robust way of dealing with
# unification
found.vars[rule_index].assign contains var if {
	some rule_index, calls in found.calls
	some call in calls

	call[0].value[0].type == "var"
	call[0].value[0].value in {"assign", "eq"}

	some var in _find_assign_vars(call[1])
}

found.vars[rule_index].somein contains var if {
	some rule_index, symbols in found.symbols
	some value in symbols

	value[0].type == "call"

	arr := value[0].value

	some var in array.flatten([_find_nested_vars(arr[1]), [v |
		count(arr) == 4
		some v in _find_nested_vars(arr[2])
	]])
}

found.vars[rule_index].some contains term if {
	some rule_index, symbols in found.symbols
	some value in symbols

	value[0].type != "call"

	some term in value
	term.type == "var"
}

# METADATA
# description: all refs found in module
found.refs[rule_index] contains value if {
	some i, rule_index in rule_index_strings

	walk(_rules[i], [_, value])

	value.type == "ref"
}

# METADATA
# description: all calls found in module
found.calls[rule_index] contains value if {
	some i, rule_index in rule_index_strings

	walk(_rules[i], [_, value])

	value[0].type == "ref"
}

# METADATA
# description: all symbols found in module
found.symbols[rule_index] contains value.symbols if {
	some i, rule_index in rule_index_strings

	walk(_rules[i], [_, value])
}

# METADATA
# description: all comprehensions found in module
found.comprehensions[rule_index] contains value if {
	some i, rule_index in rule_index_strings

	walk(_rules[i], [_, value])

	value.type in {"arraycomprehension", "objectcomprehension", "setcomprehension"}
}

# METADATA
# description: set containing all expressions in input AST
found.expressions[rule_index] contains value if {
	some i, rule_index in rule_index_strings
	some node in ["head", "body", "else"]

	walk(_rules[i][node], [_, value])

	value.terms
}

# METADATA
# description: |
#   answers whether a variable of the given name (value) is declared in the
#   local scope of the provided rule at the provided location
is_in_local_scope(rule, location, value) if {
	some var
	found.vars[_rule_index(rule)][_][var].value == value
	var.location != location

	_before_location(rule.head, var, util.to_location_object(location))
}

# METADATA
# description: |
#   finds all vars declared in `rule` *before* the `location` provided
#   note: this isn't 100% accurate, as it doesn't take into account `=`
#   assignments / unification, but it's likely good enough since other rules
#   recommend against those
find_vars_in_local_scope(rule, location) := [var |
	some var
	found.vars[_rule_index(rule)][_][var]

	not startswith(var.value, "$")
	_before_location(rule.head, var, util.to_location_object(location))
]

# special case — the value location of the rule head "sees"
# all local variables declared in the rule body
# regal ignore:narrow-argument
_before_location(head, _, loc) if {
	value_start := util.to_location_object(head.value.location)

	loc.row >= value_start.row
	loc.col >= value_start.col
	loc.row <= value_start.row + strings.count(value_start.text, "\n")
	loc.col <= value_start.col + count(regex.replace(value_start.text, `.*\n`, ""))
}

# regal ignore:narrow-argument
_before_location(_, var, loc) if util.to_location_object(var.location).row < loc.row

_before_location(_, var, loc) if {
	var_loc := util.to_location_object(var.location)

	var_loc.row == loc.row
	var_loc.col < loc.col
}

# METADATA
# description: find *only* names in the local scope, and not e.g. rule names
find_names_in_local_scope(rule, location) := {var.value |
	some var in find_vars_in_local_scope(rule, util.to_location_object(location))
}

# METADATA
# description: |
#   similar to `find_vars_in_local_scope`, but returns all variable names in scope
#   of the given location *and* the rule names present in the scope (i.e. module)
find_names_in_scope(rule, location) := (rule_names | imported_identifiers) | find_names_in_local_scope(
	rule,
	util.to_location_object(location),
)

# METADATA
# description: |
#   find all variables declared via `some` declarations (and *not* `some .. in`)
#   in the scope of the given location
find_some_decl_names_in_scope(rule, location) := {some_var.value |
	loc := util.to_location_object(location)
	some some_var in found.vars[_rule_index(rule)].some
	_before_location(rule.head, some_var, loc)
}
