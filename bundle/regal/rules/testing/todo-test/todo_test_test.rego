package regal.rules.testing["todo-test_test"]

import data.regal.rules.testing["todo-test"] as rule

test_fail_todo_test if {
	ast := regal.parse_module("foo_test.rego", `
	package foo_test

	todo_test_foo if { false }

	test_bar if { true }
	`)
	r := rule.report with input as ast

	r == {{
		"category": "testing",
		"description": "TODO test encountered",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/testing/todo-test",
		}],
		"title": "todo-test",
		"location": {
			"col": 2,
			"file": "foo_test.rego",
			"row": 4,
			"end": {
				"col": 15,
				"row": 4,
			},
			"text": "\ttodo_test_foo if { false }",
		},
		"level": "error",
	}}
}
