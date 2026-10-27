package regal.rules.bugs["inconsistent-rule-type_test"]

import data.regal.ast

import data.regal.rules.bugs["inconsistent-rule-type"] as rule

test_inconsistent_rule_type_default_boolean_and_string if {
	module := ast.policy(`
default hello := false

hello := "world" if input.message == "world"
`)

	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Inconsistent rule head value types",
		"level": "error",
		"location": {
			"col": 1,
			"end": {
				"col": 17,
				"row": 6,
			},
			"file": "policy.rego",
			"row": 6,
			"text": "hello := \"world\" if input.message == \"world\"",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/inconsistent-rule-type",
		}],
		"title": "inconsistent-rule-type",
	}}
}

test_inconsistent_rule_type_boolean_and_number if {
	module := ast.policy(`
default count_limit := false

count_limit := 10 if input.authenticated
`)

	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Inconsistent rule head value types",
		"level": "error",
		"location": {
			"col": 1,
			"end": {
				"col": 18,
				"row": 6,
			},
			"file": "policy.rego",
			"row": 6,
			"text": "count_limit := 10 if input.authenticated",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/inconsistent-rule-type",
		}],
		"title": "inconsistent-rule-type",
	}}
}

test_consistent_rule_type_same_scalar if {
	module := ast.policy(`
default status := "pending"

status := "ok" if input.ready
`)

	r := rule.report with input as module

	r == set()
}

test_consistent_rule_type_boolean if {
	module := ast.policy(`
default allow := false

allow if input.admin
`)

	r := rule.report with input as module

	r == set()
}

test_partial_set_rules_ignored if {
	module := ast.policy(`
items contains "a" if input.x
items contains 1 if input.y
`)

	r := rule.report with input as module

	r == set()
}
