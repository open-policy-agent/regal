package regal.rules.idiomatic["custom-in-construct_test"]

import data.regal.ast

import data.regal.rules.idiomatic["custom-in-construct"] as rule

test_fail_custom_in if {
	r := rule.report with input as ast.policy(`has(item, coll) if {
    		item == coll[_]
    }`)

	r == {{
		"category": "idiomatic",
		"description": "Custom function may be replaced by `in` keyword",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 16,
				"row": 3,
			},
			"text": "has(item, coll) if {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-in-construct",
		}],
		"title": "custom-in-construct",
	}}
}

test_fail_custom_in_reversed if {
	r := rule.report with input as ast.policy(`has(item, coll) if { coll[_] == item }`)

	r == {{
		"category": "idiomatic",
		"description": "Custom function may be replaced by `in` keyword",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 16,
				"row": 3,
			},
			"text": "has(item, coll) if { coll[_] == item }",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-in-construct",
		}],
		"title": "custom-in-construct",
	}}
}
