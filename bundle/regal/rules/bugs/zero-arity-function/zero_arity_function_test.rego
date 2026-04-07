package regal.rules.bugs["zero-arity-function_test"]

import data.regal.ast

import data.regal.rules.bugs["zero-arity-function"] as rule

test_fail_zero_arity_function if {
	r := rule.report with input as ast.policy("f() := true")

	r == {{
		"category": "bugs",
		"description": "Avoid functions without args",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 2,
				"row": 3,
			},
			"text": "f() := true",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/zero-arity-function",
		}],
		"title": "zero-arity-function",
	}}
}

test_fail_zero_arity_nested_function if {
	r := rule.report with input as ast.policy("a.b.c() := true")

	r == {{
		"category": "bugs",
		"description": "Avoid functions without args",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 6,
				"row": 3,
			},
			"text": "a.b.c() := true",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/zero-arity-function",
		}],
		"title": "zero-arity-function",
	}}
}

test_success_not_zero_arity_function if {
	module := ast.policy("f(_) := true")

	r := rule.report with input as module
	r == set()
}
