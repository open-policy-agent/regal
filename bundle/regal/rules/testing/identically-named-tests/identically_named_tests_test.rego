package regal.rules.testing["identically-named-tests_test"]

import data.regal.rules.testing["identically-named-tests"] as rule

test_fail_identically_named_tests if {
	ast := regal.parse_module("foo_test.rego", `
	package foo_test

	test_foo if { false }
	test_foo if { true }
	`)
	r := rule.report with input as ast

	r == {{
		"category": "testing",
		"description": "Multiple tests with same name",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/testing/identically-named-tests",
		}],
		"title": "identically-named-tests",
		"location": {
			"row": 5,
			"col": 2,
			"end": {
				"col": 10,
				"row": 5,
			},
			"file": "foo_test.rego",
			"text": "\ttest_foo if { true }",
		},
		"level": "error",
	}}
}

test_success_differently_named_tests if {
	ast := regal.parse_module("foo_test.rego", `
	package foo_test

	test_foo if { false }
	test_bar if { true }
	test_baz if { 1 == 1 }
	`)
	r := rule.report with input as ast

	r == set()
}
