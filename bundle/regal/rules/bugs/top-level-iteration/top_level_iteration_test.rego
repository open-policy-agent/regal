package regal.rules.bugs["top-level-iteration_test"]

import data.regal.ast

import data.regal.rules.bugs["top-level-iteration"] as rule

test_fail_top_level_iteration_wildcard if {
	r := rule.report with input as ast.with_rego_v1(`x := input.foo.bar[_]`)

	r == {{
		"category": "bugs",
		"description": "Iteration in top-level assignment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 22,
				"row": 5,
			},
			"text": "x := input.foo.bar[_]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/top-level-iteration",
		}],
		"title": "top-level-iteration",
		"level": "error",
	}}
}

test_fail_top_level_iteration_named_var if {
	r := rule.report with input as ast.with_rego_v1(`x := input.foo.bar[i]`)

	r == {{
		"category": "bugs",
		"description": "Iteration in top-level assignment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 22,
				"row": 5,
			},
			"text": "x := input.foo.bar[i]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/top-level-iteration",
		}],
		"title": "top-level-iteration",
		"level": "error",
	}}
}

test_success_top_level_known_var_ref if {
	r := rule.report with input as ast.policy(`
	i := "foo"
	x := input.foo.bar[i]`)

	r == set()
}

# https://github.com/open-policy-agent/regal/issues/852
test_success_top_level_ref_head_vars_assignment if {
	r := rule.report with input as ast.policy(`foo[x] := input[x] if some x in [1, 2, 3]`)

	r == set()
}

# https://github.com/open-policy-agent/regal/issues/401
test_success_top_level_input_assignment if {
	r := rule.report with input as ast.policy(`x := input`)

	r == set()
}

test_success_top_level_input_ref if {
	r := rule.report with input as ast.policy(`x := input.foo.bar[input.y]`)

	r == set()
}

test_success_top_level_const if {
	r := rule.report with input as ast.policy(`x := input.foo.bar[4]`)

	r == set()
}

test_success_top_level_param if {
	r := rule.report with input as ast.policy(`x(y) := input.foo.bar[y]`)

	r == set()
}

test_success_top_level_import if {
	r := rule.report with input as ast.policy(`
	import data.x

	y := input[x]`)

	r == set()
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1277
test_success_vars_in_head_also_in_body if {
	r := rule.report with input as ast.policy(`x := input.x[a][b] if [a, b] := [1, 2]`)

	r == set()
}
