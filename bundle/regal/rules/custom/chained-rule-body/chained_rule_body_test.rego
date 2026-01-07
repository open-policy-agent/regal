package regal.rules.custom["chained-rule-body_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.custom["chained-rule-body"] as rule

test_fail_chained_incremental_definition if {
	module := ast.policy(`rule if {
		input.x
	} {
		input.y
	}`)
	r := rule.report with input as module

	r == {{
		"category": "custom",
		"description": "Avoid chaining rule bodies",
		"level": "error",
		"location": {
			"col": 4,
			"file": "policy.rego",
			"row": 5, "text": "\t} {",
			"end": {
				"col": 3,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/chained-rule-body", "custom"),
		}],
		"title": "chained-rule-body",
	}}
}

test_success_not_chained_incremental_definition if {
	module := ast.policy(`
	rule if {
		input.x
	}

	rule if {
		input.y
	}`)

	r := rule.report with input as module
	r == set()
}
