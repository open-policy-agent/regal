package regal.rules.style["trailing-default-rule_test"]

import data.regal.ast

import data.regal.rules.style["trailing-default-rule"] as rule

test_success_default_declared_first if {
	module := ast.policy(`
	default foo := true

	foo if true
	`)
	r := rule.report with input as module

	r == set()
}

test_fail_default_declared_after if {
	module := ast.policy(`
	foo if true

	default foo := true
	`)
	r := rule.report with input as module

	r == {{
		"category": "style",
		"description": "Default rule should be declared first",
		"level": "error",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 6,
			"end": {
				"col": 9,
				"row": 6,
			},
			"text": "\tdefault foo := true",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/trailing-default-rule",
		}],
		"title": "trailing-default-rule",
	}}
}
