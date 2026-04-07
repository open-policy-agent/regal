package regal.rules.style["todo-comment_test"]

import data.regal.ast

import data.regal.rules.style["todo-comment"] as rule

test_fail_todo_comment if {
	r := rule.report with input as ast.policy(`# TODO: do something clever`)

	r == {{
		"category": "style",
		"description": "Avoid TODO comments",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/todo-comment",
		}],
		"title": "todo-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 28,
				"row": 3,
			},
			"text": `# TODO: do something clever`,
		},
		"level": "error",
	}}
}

test_fail_fixme_comment if {
	r := rule.report with input as ast.policy(`# fixme: this is broken`)

	r == {{
		"category": "style",
		"description": "Avoid TODO comments",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/todo-comment",
		}],
		"title": "todo-comment",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 24,
				"row": 3,
			},
			"text": `# fixme: this is broken`,
		},
		"level": "error",
	}}
}

test_success_no_todo_comment if {
	r := rule.report with input as ast.policy(`# This code is great`)
	r == set()
}
