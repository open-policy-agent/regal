package regal.rules.idiomatic["equals-pattern-matching_test"]

import data.regal.ast

import data.regal.rules.idiomatic["equals-pattern-matching"] as rule

test_fail_simple_head_comparison_could_be_matched_in_arg if {
	r := rule.report with input as ast.policy("f(x) := x == 1")

	r == expected_with_location({
		"col": 1,
		"row": 3,
		"end": {
			"col": 15,
			"row": 3,
		},
		"text": "f(x) := x == 1",
	})
}

test_fail_simple_head_comparison_could_be_matched_in_arg_eq_order if {
	r := rule.report with input as ast.policy("f(x) := 1 == x")

	r == expected_with_location({
		"col": 1,
		"row": 3,
		"end": {
			"col": 15,
			"row": 3,
		},
		"text": "f(x) := 1 == x",
	})
}

test_fail_simple_head_comparison_could_be_matched_in_arg_multiple_args if {
	r := rule.report with input as ast.policy("f(_, x, _) := x == 1")

	r == expected_with_location({
		"col": 1,
		"row": 3,
		"end": {
			"col": 21,
			"row": 3,
		},
		"text": "f(_, x, _) := x == 1",
	})
}

test_fail_simple_body_comparison_could_be_matched_in_arg if {
	r := rule.report with input as ast.policy(`f(x) := "one" if {
		x == 1
	}`)

	r == expected_with_location({
		"col": 1,
		"row": 3,
		"end": {
			"col": 3,
			"row": 5,
		},
		"text": "f(x) := \"one\" if {",
	})
}

test_fail_simple_body_comparison_could_be_matched_in_arg_eq_order if {
	r := rule.report with input as ast.policy(`f(x) := "one" if {
		1 == x
	}`)

	r == expected_with_location({
		"col": 1,
		"row": 3,
		"end": {
			"col": 3,
			"row": 5,
		},
		"text": "f(x) := \"one\" if {",
	})
}

test_fail_simple_body_comparison_could_be_matched_using_if if {
	r := rule.report with input as ast.with_rego_v1(`f(x) := x if x == 1`)

	r == expected_with_location({
		"col": 1,
		"row": 5,
		"end": {
			"col": 20,
			"row": 5,
		},
		"text": "f(x) := x if x == 1",
	})
}

test_success_actually_pattern_matching if {
	module := ast.policy("f(1)")
	r := rule.report with input as module

	r == set()
}

test_success_skipped_on_else if {
	module := ast.policy(`f(x) if {
		x == 1
	} else := false`)
	r := rule.report with input as module

	r == set()
}

expected := {
	"category": "idiomatic",
	"description": "Prefer pattern matching in function arguments",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/equals-pattern-matching",
	}],
	"title": "equals-pattern-matching",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
