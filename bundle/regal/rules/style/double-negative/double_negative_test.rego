package regal.rules.style["double-negative_test"]

import data.regal.ast

import data.regal.rules.style["double-negative"] as rule

test_fail_double_negative if {
	r := rule.report with input as ast.policy(`
    import future.keywords.if

    not_fine := true

    fine if not not_fine
    `)

	r == {{
		"category": "style",
		"description": "Avoid double negatives",
		"location": {
			"col": 13,
			"file": "policy.rego",
			"row": 8, "text": "    fine if not not_fine",
			"end": {
				"col": 25,
				"row": 8,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/double-negative",
		}],
		"title": "double-negative",
		"level": "error",
	}}
}
