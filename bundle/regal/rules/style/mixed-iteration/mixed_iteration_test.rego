package regal.rules.style["mixed-iteration_test"]

import data.regal.ast

import data.regal.rules.style["mixed-iteration"] as rule

test_fail_mixed_iteration if {
	r := rule.report with input as ast.policy("fail if some x in input[_]")

	r == {{
		"category": "style",
		"description": "Mixed iteration style",
		"level": "error",
		"location": {
			"col": 19,
			"end": {
				"col": 27,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "fail if some x in input[_]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/mixed-iteration",
		}],
		"title": "mixed-iteration",
	}}
}

test_fail_mixed_iteration_nested if {
	r := rule.report with input as ast.policy("fail if some x in input[_].y[_]")

	r == {{
		"category": "style",
		"description": "Mixed iteration style",
		"level": "error",
		"location": {
			"col": 19,
			"end": {
				"col": 32,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "fail if some x in input[_].y[_]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/mixed-iteration",
		}],
		"title": "mixed-iteration",
	}}
}

test_success_not_an_ouput_var if {
	r := rule.report with input as ast.policy(`
	success if {
		y := 1
		some x in input[y]
	}`)

	r == set()
}
