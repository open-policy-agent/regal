package regal.rules.style["unnecessary-some_test"]

import data.regal.ast

import data.regal.rules.style["unnecessary-some"] as rule

test_fail_some_unnecessary_value if {
	module := ast.with_rego_v1(`
	rule if {
		some "x" in ["x"]
	}
	`)
	r := rule.report with input as module

	r == {{
		"category": "style",
		"description": "Unnecessary use of `some`",
		"level": "error",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 7,
			"end": {
				"col": 20,
				"row": 7,
			},
			"text": "\t\tsome \"x\" in [\"x\"]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/unnecessary-some",
		}],
		"title": "unnecessary-some",
	}}
}

test_fail_some_unnecessary_key_value if {
	r := rule.report with input as ast.with_rego_v1(`rule if some "x", 1 in {"x": 1}`)

	r == {{
		"category": "style",
		"description": "Unnecessary use of `some`",
		"level": "error",
		"location": {
			"col": 14,
			"end": {
				"col": 32,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5,
			"text": "rule if some \"x\", 1 in {\"x\": 1}",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/unnecessary-some",
		}],
		"title": "unnecessary-some",
	}}
}

test_success_some_value_using_var if {
	r := rule.report with input as ast.policy(`rule if some var in input.vars`)

	r == set()
}

test_success_some_key_value_using_var_for_value if {
	r := rule.report with input as ast.policy(`rule if some "x", var in {"x": 1}`)

	r == set()
}

test_success_some_key_value_using_var_for_key if {
	r := rule.report with input as ast.policy(`rule if some var, 1 in {"x": 1}`)

	r == set()
}

test_success_just_in_head if {
	r := rule.report with input as ast.policy(`rule := [1 in []]`)

	r == set()
}
