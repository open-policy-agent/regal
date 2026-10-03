package regal.rules.bugs["impossible-default_test"]

import data.regal.ast

import data.regal.rules.bugs["impossible-default"] as rule

test_fail_default_with_unconditional_definition if {
	r := rule.report with input as ast.policy(`
	default username := "guest"

	username := "regal"
	`)

	r == {{
		"category": "bugs",
		"description": "Rule defined unconditionally alongside default definition",
		"level": "error",
		"location": {
			"col": 2,
			"end": {
				"col": 9,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\tdefault username := \"guest\"",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/impossible-default",
		}],
		"title": "impossible-default",
	}}
}

test_fail_default_with_unconditional_definition_same_values if {
	r := rule.report with input as ast.policy(`
	default allow := false

	allow := false
	`)

	_flagged(r) == {"\tdefault allow := false"}
}

test_fail_default_declared_after_unconditional if {
	r := rule.report with input as ast.policy(`
	username := "regal"

	default username := "guest"
	`)

	_flagged(r) == {"\tdefault username := \"guest\""}
}

test_fail_default_alongside_both_conditional_and_unconditional if {
	r := rule.report with input as ast.policy(`
	default username := "guest"

	username := "admin" if input.admin

	username := "regal"
	`)

	_flagged(r) == {"\tdefault username := \"guest\""}
}

test_fail_default_ref_head if {
	r := rule.report with input as ast.policy(`
	default config.mode := "strict"

	config.mode := "loose"
	`)

	_flagged(r) == {"\tdefault config.mode := \"strict\""}
}

test_fail_default_function_with_wildcard_arg if {
	r := rule.report with input as ast.policy(`
	default f(_) := "guest"

	f(_) := "regal"
	`)

	_flagged(r) == {"\tdefault f(_) := \"guest\""}
}

test_fail_default_function_with_named_arg if {
	r := rule.report with input as ast.policy(`
	default f(_) := "guest"

	f(x) := "regal"
	`)

	_flagged(r) == {"\tdefault f(_) := \"guest\""}
}

test_fail_default_function_with_multiple_args if {
	r := rule.report with input as ast.policy(`
	default f(_, _) := "guest"

	f(x, y) := "regal"
	`)

	_flagged(r) == {"\tdefault f(_, _) := \"guest\""}
}

test_fail_multiple_defaults_different_names if {
	r := rule.report with input as ast.policy(`
	default username := "guest"
	default role := "viewer"

	username := "regal"
	role := "admin" if input.admin
	`)

	_flagged(r) == {"\tdefault username := \"guest\""}
}

test_success_default_with_only_conditional_definition if {
	r := rule.report with input as ast.policy(`
	default username := "guest"

	username := "admin" if input.admin
	`)

	r == set()
}

test_success_default_with_non_constant_value if {
	r := rule.report with input as ast.policy(`
	default username := "guest"

	username := input.user
	`)

	r == set()
}

test_success_default_function_with_literal_arg if {
	r := rule.report with input as ast.policy(`
	default f(_) := "guest"

	f("admin") := "boss"
	`)

	r == set()
}

test_success_default_function_with_repeated_var_arg if {
	r := rule.report with input as ast.policy(`
	default f(_, _) := "guest"

	f(x, x) := "regal"
	`)

	r == set()
}

test_success_default_function_with_different_arity if {
	r := rule.report with input as ast.policy(`
	default f(_) := "guest"

	f(x, y) := "regal"
	`)

	r == set()
}

test_success_default_only if {
	r := rule.report with input as ast.policy(`
	default username := "guest"
	`)

	r == set()
}

test_success_no_default if {
	r := rule.report with input as ast.policy(`
	username := "regal"
	`)

	r == set()
}

test_success_default_with_else_chain if {
	r := rule.report with input as ast.policy(`
	default allow := false

	allow := true if input.admin else := false if input.user
	`)

	r == set()
}

test_success_dynamic_ref_head if {
	r := rule.report with input as ast.policy(`
	default p := "default"

	p[input.key] := "value"
	`)

	r == set()
}

_flagged(reports) := {report.location.text | some report in reports}
