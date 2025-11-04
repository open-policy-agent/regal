package regal.rules.testing["test-outside-test-package_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.testing["test-outside-test-package"] as rule

test_fail_test_outside_test_package if {
	r := rule.report with input as ast.policy(`test_foo if { false }`) with input.regal.file.name as "p_test.rego"

	r == {{
		"category": "testing",
		"description": "Test outside of test package",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/test-outside-test-package", "testing"),
		}],
		"title": "test-outside-test-package",
		"location": {
			"col": 1,
			"file": "p_test.rego",
			"row": 3,
			"end": {"col": 9, "row": 3},
			"text": `test_foo if { false }`,
		},
		"level": "error",
	}}
}

test_success_test_inside_test_package if {
	ast := regal.parse_module("foo_test.rego", `
	package foo_test

	test_foo if { false }
	`)
	result := rule.report with input as ast
	result == set()
}

test_success_test_inside_test_package_named_just_test if {
	ast := regal.parse_module("test.rego", `
	package test

	test_foo if { false }
	`)
	result := rule.report with input as ast
	result == set()
}

# https://github.com/open-policy-agent/regal/issues/176
test_success_test_prefixed_function if {
	ast := regal.parse_module("foo_test.rego", `
	package foo

	test_foo(x) if { x == 1 }
	`)
	result := rule.report with input as ast
	result == set()
}
