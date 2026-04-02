package regal.rules.style["default-over-not_test"]

import data.regal.ast

import data.regal.rules.style["default-over-not"] as rule

test_fail_default_over_not if {
	r := rule.report with input as ast.with_rego_v1(`
	user := input.user
	user := "foo" if not input.user
	`)

	r == {{
		"category": "style",
		"description": "Prefer default assignment over negated condition",
		"level": "error",
		"location": {
			"col": 19,
			"file": "policy.rego",
			"row": 7,
			"end": {
				"col": 33,
				"row": 7,
			},
			"text": "\tuser := \"foo\" if not input.user",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/default-over-not",
		}],
		"title": "default-over-not",
	}}
}

test_success_non_constant_value if {
	r := rule.report with input as ast.policy(`
	user := input.user
	user := var if not input.user
	`)

	r == set()
}

test_success_var_in_ref if {
	r := rule.report with input as ast.policy(`
	user := input[x].user
	user := "foo" if not input[x].user
	`)

	r == set()
}
