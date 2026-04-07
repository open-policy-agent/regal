package regal.rules.idiomatic["in-wildcard-key_test"]

import data.regal.ast

import data.regal.rules.idiomatic["in-wildcard-key"] as rule

test_fail_wildcard_key_not_needed if {
	r := rule.report with input as ast.policy("r if some _, v in input")

	r == {{
		"category": "idiomatic",
		"description": "Unnecessary wildcard key",
		"level": "error",
		"location": {
			"col": 11,
			"end": {
				"col": 12,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r if some _, v in input",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/in-wildcard-key",
		}],
		"title": "in-wildcard-key",
	}}
}

test_success[case] if {
	some case in [
		"r if some v in input",
		"r if some k, _ in input",
		"r if some [k], v in input",
		"r if some [_, k], v in input",
	]

	r := rule.report with input as ast.policy(case)
	r == set()
}
