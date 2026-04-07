package regal.rules.bugs["if-empty-object_test"]

import data.regal.ast

import data.regal.rules.bugs["if-empty-object"] as rule

test_fail_if_empty_object if {
	r := rule.report with input as ast.policy("rule if {}")

	r == {{
		"category": "bugs",
		"description": "Empty object following `if`",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"text": "rule if {}",
			"end": {
				"col": 11,
				"row": 3,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/if-empty-object",
		}],
		"title": "if-empty-object",
	}}
}

test_success_if_non_empty_object if {
	# this is arguably just as useless, but we'll defer
	# to the constant-condition rule for these cases
	r := rule.report with input as ast.policy(`rule if {"foo": "bar"}`)

	r == set()
}
