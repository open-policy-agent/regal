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
#   find vars like input[x].foo[y] where x and y are vars
#   note: value.type == "ref" check must have been done before calling this function
find_ref_vars(value) := [var | # regal ignore:narrow-argument
	some i, var in value.value

	i > 0
	var.type == "var"
]

# one or two vars declared via `every`, i.e. `every x in y {}`
# or `every`, i.e. `every x, y in z {}`
_find_every_vars(value) := [var |
	some kind in ["key", "value"]

	var := value[kind]

	var.type == "var"
	indexof(var.value, "$") == -1
]

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

# find "return value vars", i.e. those placed after a functions declared
# number of args, e.g `x` in `count(x, y)`
_find_vars(value, last) := {"term": find_term_vars(undeclared)} if {
	last == "terms"
	value[0].type == "ref"
	value[0].value[0].type == "var"
	value[0].value[0].value != "assign"

	fn_name := ref_static_to_string(value[0].value)
	undeclared_start := count(all_functions[fn_name].decl.args) + 1

	value[undeclared_start]
	fn_name != "print"

	undeclared := array.slice(value, undeclared_start, 100)
}

# `=` isn't necessarily assignment, and only considering the variable on the
# left-hand side is equally dubious, but we'll treat `x = 1` as `x := 1` for
# the purpose of this function until we have a more robust way of dealing with
# unification
_find_vars(value, last) := {"assign": _find_assign_vars(value[1])} if {
	last == "terms"
	value[0].type == "ref"
	value[0].value[0].type == "var"
	value[0].value[0].value in {"assign", "eq"}
}

_find_vars(value, last) := {"every": _find_every_vars(value)} if {
	last == "terms"
	value.domain
}

_find_vars(value, last) := {"somein": vars} if {
	last == "symbols"
	value[0].type == "call"

	arr := value[0].value

	vars := array.flatten([_find_nested_vars(arr[1]), [v |
		count(arr) == 4
		some v in _find_nested_vars(arr[2])
	]])
}

_find_vars(value, last) := {"some": vars} if {
	last == "symbols"
	value[0].type != "call"

	vars := [v |
		some v in value
		v.type == "var"
	]
}

_find_vars(value, last) := {"args": arg_vars} if {
	last == "args"

	arg_vars := [arg |
		some arg in value
		arg.type == "var"
	]

	arg_vars != []
}

# converting to string until https://github.com/open-policy-agent/opa/issues/6736 is fixed
_rule_index(rule) := rule_index_strings[i] if {
	some i, r in _rules
	r == rule
}

# METADATA
# description: |
#   traverses all nodes under provided node (using `walk`), and returns an array with
#   all variables declared via assignment (:=), `some`, `every` and in comprehensions
#   DEPRECATED: uses ast.found.vars instead
find_vars(node) := array.concat(
	[var |
		walk(node, [path, value])

		last := {"terms", "symbols", "args"}[regal.last(path)]
		var := _find_vars(value, last)[_][_]
	],
	[var |
		walk(node, [_, value])

		value.type == "ref"

		some x, var in value.value
		x > 0
		var.type == "var"
	],
)

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
found.vars[rule_index][context] contains var if {
	some rule_index in rule_index_strings

	terms := found.expressions[rule_index][_].terms

	some context, vars in _find_vars(terms, "terms")
	some var in vars
}

found.vars[rule_index][context] contains var if {
	some i, rule_index in rule_index_strings
	some expr in _rules[i].body

	walk(expr.terms, [_, value])

	some context, vars in _find_vars(value.symbols, "symbols")
	some var in vars
}

found.vars[rule_index][context] contains var if {
	some i, rule_index in rule_index_strings
	rule := _rules[i].else
	some node in ["head", "body", "else"]

	walk(rule[node], [_, value])

	some context, vars in _find_vars(value.symbols, "symbols")
	some var in vars
}

found.vars[rule_index][context] contains var if {
	some i, rule_index in rule_index_strings
	head := _rules[i].head

	some node in ["key", "value"]
	not head[node].type in {"string", "number", "boolean", "null", "var"}

	walk(head[node].value, [path, value])

	last := {"terms", "symbols"}[regal.last(path)]

	some context, vars in _find_vars(value, last)
	some var in vars
}

found.vars[rule_index].args contains term if {
	some i, rule_index in rule_index_strings
	some term in _rules[i].head.args

	term.type == "var"
}

found.vars[rule_index].args contains value if {
	some i, rule_index in rule_index_strings
	some term in _rules[i].head.args

	term.type == "array" # only composite type that can contain vars in args position (right?)

	some item in term.value
	not item.type in {"string", "number", "boolean", "null"}

	walk(item, [_, value])

	value.type == "var"
}

found.vars[rule_index].ref contains term if {
	some rule_index in rule_index_strings
	some ref in found.refs[rule_index]
	some x, term in ref.value

	x > 0
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
