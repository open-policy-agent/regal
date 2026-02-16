package regal.rules.performance["equals-over-count_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.performance["equals-over-count"] as rule

test_fail_count_equals_zero if {
	r := rule.report with input as ast.policy("r if count(input.rules) == 0")

	r == {{
		"category": "performance",
		"description": "Add description of rule here!",
		"level": "error",
		"location": {
			"col": 25,
			"end": {
				"col": 27,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r if count(input.rules) == 0",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/equals-over-count", "performance"),
		}],
		"title": "equals-over-count",
	}}
}
