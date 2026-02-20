package regal.rules.performance["equals-over-count_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.performance["equals-over-count"] as rule

test_fail_count_equals_zero[text] if {
	some text in [
		"r if count(input.rules) == 0",
		"r if count(input.rules) != 0",
		"r if count(input.rules) >  0",
	]

	r := rule.report with input as ast.policy(text)

	r == {{
		"category": "performance",
		"description": "Prefer direct use of `==`/`!=` over `count` to check for empty collections",
		"level": "error",
		"location": {
			"col": 6,
			"end": {
				"col": 29,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": text,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/equals-over-count", "performance"),
		}],
		"title": "equals-over-count",
	}}
}

test_success_count_not_compared_to_zero[text] if {
	some text in [
		"r if count(input.rules) == 1",
		"r if count(input.rules) != 2",
		"r if count(input.rules) <  1",
	]

	r := rule.report with input as ast.policy(text)

	r == set()
}
