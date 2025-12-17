package regal.ast_test

import data.regal.ast
import data.regal.capabilities
import data.regal.config

# https://github.com/open-policy-agent/regal/issues/168
test_function_decls_multiple_same_name if {
	policy := `package p

	f(x) := x if true
	f(y) := y if false
	`
	module := regal.parse_module("p.rego", policy)
	custom := ast.function_decls with input as module

	# we only need to assert there wasn't a conflict in the above
	# call, not what value was returned
	is_object(custom)
}

test_comment_blocks if {
	policy := `package p

# METADATA
# title: foo
# bar: invalid
allow := true

# not metadata

# another
# block
`

	module := regal.parse_module("p.rego", policy)
	blocks := ast.comment_blocks(module.comments) with input as module
	blocks == [
		[
			{"location": "3:1:5:15", "text": "IE1FVEFEQVRB"},
			{"location": "4:1:4:13", "text": "IHRpdGxlOiBmb28="},
			{"location": "5:1:5:15", "text": "IGJhcjogaW52YWxpZA=="},
		],
		[{"location": "8:1:8:15", "text": "IG5vdCBtZXRhZGF0YQ=="}],
		[
			{"location": "10:1:10:10", "text": "IGFub3RoZXI="},
			{"location": "11:1:11:8", "text": "IGJsb2Nr"},
		],
	]
}

test_find_names_in_scope if {
	policy := `
	package p

	bar := "baz"

	global := "foo"

	comp := [foo | foo := input[_]]

	allow if {
		a := global
		b := [c | c := input[_]]

		every d in input {
			d == "foo"
			e := "bar"
			e == "foo"
		}
	}`

	module := regal.parse_module("p.rego", policy)
	allow_rule := module.rules[3]

	in_scope := ast.find_names_in_scope(allow_rule, {"col": 1, "row": 30}) with input as module
		with config.capabilities as capabilities.provided

	in_scope == {"bar", "global", "comp", "allow", "a", "b", "c", "d", "e"}
}

var_names(vars) := {var.value | some var in vars}

test_provided_capabilities_never_undefined if capabilities.provided == {} with data.internal as {}

test_function_calls if {
	calls := ast.function_calls["0"] with input as ast.with_rego_v1(`
	rule if {
		x := 1
		f(2)
	}`)

	{"assign", "f"} == {call.name | some call in calls}
}

test_implicit_boolean_assignment if ast.implicit_boolean_assignment(ast.with_rego_v1(`a.b if true`).rules[0])

test_ref_to_string if {
	ast.ref_to_string([{"type": "var", "value": "data"}]) == `data`
	ast.ref_to_string([{"type": "var", "value": "foo"}, {"type": "var", "value": "bar"}]) == `foo[bar]`
	ast.ref_to_string([{"type": "var", "value": "data"}, {"type": "string", "value": "/foo/"}]) == `data["/foo/"]`
	ast.ref_to_string([
		{"type": "var", "value": "foo"},
		{"type": "var", "value": "bar"},
		{"type": "var", "value": "baz"},
	]) == `foo[bar][baz]`
	ast.ref_to_string([
		{"type": "var", "value": "foo"},
		{"type": "var", "value": "bar"},
		{"type": "var", "value": "baz"},
		{"type": "string", "value": "qux"},
	]) == `foo[bar][baz].qux`
	ast.ref_to_string([
		{"type": "var", "value": "foo"},
		{"type": "string", "value": "~bar~"},
		{"type": "string", "value": "boo"},
		{"type": "var", "value": "baz"},
	]) == `foo["~bar~"].boo[baz]`
	ast.ref_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "string", "value": "lsp"},
		{"type": "string", "value": "completion_test"},
	]) == `data.regal.lsp.completion_test`
	ast.ref_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "number", "value": 1},
	]) == `data.regal[1]`
	ast.ref_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "boolean", "value": true},
	]) == `data.regal[true]`
}

test_ref_static_to_string if {
	ast.ref_static_to_string([{"type": "var", "value": "data"}]) == `data`
	ast.ref_static_to_string([{"type": "var", "value": "foo"}, {"type": "var", "value": "bar"}]) == `foo`
	ast.ref_static_to_string([{"type": "var", "value": "data"}, {"type": "string", "value": "/foo/"}]) == `data["/foo/"]`
	ast.ref_static_to_string([
		{"type": "var", "value": "foo"},
		{"type": "string", "value": "bar"},
		{"type": "var", "value": "baz"},
	]) == `foo.bar`
	ast.ref_static_to_string([
		{"type": "var", "value": "foo"},
		{"type": "string", "value": "~bar~"},
		{"type": "string", "value": "qux"},
	]) == `foo["~bar~"].qux`
	ast.ref_static_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "string", "value": "lsp"},
		{"type": "string", "value": "completion_test"},
	]) == `data.regal.lsp.completion_test`
	ast.ref_static_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "number", "value": 1},
	]) == `data.regal[1]`
	ast.ref_static_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "boolean", "value": true},
	]) == `data.regal[true]`
	ast.ref_static_to_string([
		{"type": "var", "value": "data"},
		{"type": "string", "value": "regal"},
		{"type": "boolean", "value": false},
	]) == `data.regal[false]`
}

test_rule_head_locations if {
	policy := `package policy

default allow := false

allow if true

reasons contains "foo"
reasons contains "bar"

default my_func(_) := false
my_func(1) := true

ref_rule[foo] := true if {
	some foo in [1,2,3]
}
`
	result := ast.rule_head_locations with input as regal.parse_module("p.rego", policy)

	result == {
		"data.policy.allow": {{"col": 9, "row": 3}, {"col": 1, "row": 5}},
		"data.policy.reasons": {{"col": 1, "row": 7}, {"col": 1, "row": 8}},
		"data.policy.my_func": {{"col": 9, "row": 10}, {"col": 1, "row": 11}},
		"data.policy.ref_rule": {{"col": 1, "row": 13}},
	}
}

test_public_rules_and_functions if {
	module := regal.parse_module("p.rego", `package p

foo := true

_bar := false

x.y := true

x._z := false
	`)

	public := ast.public_rules_and_functions with input as module

	{ast.ref_to_string(rule.head.ref) | some rule in public} == {"foo", "x.y"}
}

test_var_in_head[case] if {
	name := "foo"

	some case, head in {
		"value": {"value": {"type": "var", "value": name}},
		"key": {"key": {"type": "var", "value": name}},
		"term value": {"value": {"type": "object", "value": [[
			{"value": "foo", "type": "string"},
			{"type": "var", "value": name},
		]]}},
		"term key": {"key": {"type": "object", "value": [[
			{"value": "foo", "type": "string"},
			{"type": "var", "value": name},
		]]}},
		"ref": {"ref": [
			{"type": "var", "value": "var"},
			{"type": "var", "value": name},
		]},
	}

	ast.var_in_head(head, "foo")
}

test_var_in_head_templatestring[case] if {
	some case, head in {
		"value": `rule := $"{x}"`,
		"key": `rule contains $"{x}"`,
		"term value": `rule := {"a": $"{x}"}`,
		"term key": `rule contains { $"{x}": "a" }`,
		"ref": `rule[$"{x}"].y if { x := 1 }`,
	}

	ast.var_in_head(ast.policy(head).rules[0].head, "x")
}

test_builtin_functions_called_in_templatestring if {
	module := ast.policy(`r := $"{upper("a")} {lower("b")}"`)
	names := ast.builtin_functions_called with input as module with config.capabilities as capabilities.provided

	names == {"upper", "lower"}
}
