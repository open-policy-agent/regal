package regal.rules.idiomatic["single-item-in_test"]

import data.regal.ast

import data.regal.rules.idiomatic["single-item-in"] as rule

test_fail_single_item_in_array if {
	r := rule.report with input as ast.policy("fail if 1 in [1]")

	r == {{
		"category": "idiomatic",
		"description": "Avoid `in` for single item collection",
		"level": "error",
		"location": {
			"col": 11,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "fail if 1 in [1]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/single-item-in",
		}],
		"title": "single-item-in",
	}}
}

test_fail_single_item_in_set if {
	r := rule.report with input as ast.policy("fail if 1 in {1}")

	r == {{
		"category": "idiomatic",
		"description": "Avoid `in` for single item collection",
		"level": "error",
		"location": {
			"col": 11,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "fail if 1 in {1}",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/single-item-in",
		}],
		"title": "single-item-in",
	}}
}

test_fail_single_item_in_object if {
	r := rule.report with input as ast.policy(`fail if 1 in {"x": 1}`)

	r == {{
		"category": "idiomatic",
		"description": "Avoid `in` for single item collection",
		"level": "error",
		"location": {
			"col": 11,
			"end": {
				"col": 13,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": `fail if 1 in {"x": 1}`,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/single-item-in",
		}],
		"title": "single-item-in",
	}}
}

test_success_in_used_on_var if {
	r := rule.report with input as ast.policy(`success if 1 in var`)

	r == set()
}

test_success_some_in_used_on_var if {
	r := rule.report with input as ast.policy(`success if [y | some y in x]`)

	r == set()
}
