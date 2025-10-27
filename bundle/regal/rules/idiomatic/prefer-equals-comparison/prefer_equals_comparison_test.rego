package regal.rules.idiomatic["prefer-equals-comparison_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.idiomatic["prefer-equals-comparison"] as rule

test_fail_impossible_assignment_2_constants if {
	# this is of course also a constant-condition violation, but no rule
	# should make assumptions about what other rules exist, or what they do
	r := rule.report with input as ast.policy("r if true = true")

	r == {{
		"category": "idiomatic",
		"description": "Prefer `==` for equality comparison",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 3,
			"col": 6,
			"end": {
				"row": 3,
				"col": 17,
			},
			"text": "r if true = true",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/prefer-equals-comparison", "idiomatic"),
		}],
		"title": "prefer-equals-comparison",
	}}
}

test_fail_impossible_assignment_static_ref_equals_constant if {
	r := rule.report with input as ast.policy("r if input.foo = 22")

	r == {{
		"category": "idiomatic",
		"description": "Prefer `==` for equality comparison",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 3,
			"col": 6,
			"end": {
				"row": 3,
				"col": 20,
			},
			"text": "r if input.foo = 22",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/prefer-equals-comparison", "idiomatic"),
		}],
		"title": "prefer-equals-comparison",
	}}
}

test_fail_impossible_assignment_constant_equals_static_ref if {
	r := rule.report with input as ast.policy("r if 42 = input.bar")

	r == {{
		"category": "idiomatic",
		"description": "Prefer `==` for equality comparison",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 3,
			"col": 6,
			"end": {
				"row": 3,
				"col": 20,
			},
			"text": "r if 42 = input.bar",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/prefer-equals-comparison", "idiomatic"),
		}],
		"title": "prefer-equals-comparison",
	}}
}

test_fail_impossible_assignment_input_var_equals_static_ref if {
	r := rule.report with input as ast.policy(`r if {
		x := 42
		x = input.bar
	}`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer `==` for equality comparison",
		"level": "error",
		"location": {
			"col": 3,
			"row": 5,
			"end": {
				"col": 16,
				"row": 5,
			},
			"file": "policy.rego",
			"text": "\t\tx = input.bar",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/prefer-equals-comparison", "idiomatic"),
		}],
		"title": "prefer-equals-comparison",
	}}
}

test_success_not_impossible_assignment_output_var_equals_static_ref if {
	r := rule.report with input as ast.policy(`r if {
		x = input.bar
	}`)

	r == set()
}
