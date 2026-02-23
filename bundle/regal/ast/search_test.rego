package regal.ast_test

import data.regal.ast

test_find_some_decl_names_in_scope if {
	policy := `package p

	allow if {
		foo := 1
		some x
		input[x]
		some y, z
		input[y][z] == x
	}`

	module := regal.parse_module("p.rego", policy)

	{"x"} == ast.find_some_decl_names_in_scope(module.rules[0], {"col": 1, "row": 6}) with input as module
	{"x", "y", "z"} == ast.find_some_decl_names_in_scope(module.rules[0], {"col": 1, "row": 8}) with input as module
}

test_find_vars_in_local_scope if {
	policy := `
	package p

	global := "foo"

	allow if {
		a := global
		b := [c | c := input[d]]

		every e in input {
			f == "foo"
			g := "bar"
			h == "foo"
		}
	}`

	module := regal.parse_module("p.rego", policy)

	allow_rule := module.rules[1]

	var_locations := {
		"a": {"col": 3, "row": 9},
		"b": {"col": 3, "row": 10},
		"c": {"col": 13, "row": 10},
		"d": {"col": 9, "row": 12},
		"e": {"col": 4, "row": 14},
	}

	var_names(ast.find_vars_in_local_scope(allow_rule, var_locations.a)) with input as module == set()
	var_names(ast.find_vars_in_local_scope(allow_rule, var_locations.b)) with input as module == {"a"}
	var_names(ast.find_vars_in_local_scope(allow_rule, var_locations.c)) with input as module == {"a", "b", "c"}
	var_names(ast.find_vars_in_local_scope(allow_rule, var_locations.d)) with input as module == {"a", "b", "c", "d"}
	var_names(ast.find_vars_in_local_scope(allow_rule, var_locations.e)) with input as module == {"a", "b", "c", "d", "e"}
}

test_find_vars_in_local_scope_complex_comprehension_term if {
	policy := `
	package p

	allow if {
		a := [{"b": b} | c := input[b]]
	}`

	module := regal.parse_module("p.rego", policy)

	allow_rule := module.rules[0]

	ast.find_vars_in_local_scope(allow_rule, {"col": 10, "row": 10}) with input as module == [
		{"location": {"col": 3, "row": 7, "text": "YQ=="}, "type": "var", "value": "a"},
		{"location": {"col": 15, "row": 7, "text": "Yg=="}, "type": "var", "value": "b"},
		{"location": {"col": 20, "row": 7, "text": "Yw=="}, "type": "var", "value": "c"},
		{"location": {"col": 31, "row": 7, "text": "Yg=="}, "type": "var", "value": "b"},
	]
}

test_found_refs_in_template_strings if {
	refs := ast.found.refs["0"] with input as ast.policy(`r := $"{input.foo + input.bar} {data.baz}"`)

	count(refs) == 4
}

test_found_calls_in_template_strings if {
	calls := ast.found.calls["0"] with input as ast.policy("r := $`{count(split(input.ref, \".\"))}`")

	count(calls) == 2
}

test_found_expressions_in_template_strings if {
	exprs := ast.found.expressions["0"] with input as ast.policy(`r if $"{x > 10}" == "true"`)

	count(exprs) == 2
	count([1 | exprs[_].interpolated]) == 1
}

test_found_comprehensions_in_template_strings if {
	comps := ast.found.comprehensions["0"] with input as ast.policy(`r := $"{[x | some x in input.arr]}"`)

	count(comps) == 1
}

test_found_symbols_in_template_strings if {
	syms := ast.found.symbols["0"] with input as ast.policy(`r := $"{[{x, y} |
		some x
		some y in input.arr
		data.foo[a][b] == x + y
	]}"`)

	count(syms) == 2
}
