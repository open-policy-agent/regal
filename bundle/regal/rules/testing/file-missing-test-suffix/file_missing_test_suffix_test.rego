package regal.rules.testing["file-missing-test-suffix_test"]

import data.regal.rules.testing["file-missing-test-suffix"] as rule

test_fail_test_in_file_without_test_suffix if {
	ast := regal.parse_module("policy.rego", `package foo_test

	test_foo if { false }
	`)

	r := rule.report with input as ast
	r == {{
		"category": "testing",
		"description": "Files containing tests should have a _test.rego suffix",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/testing/file-missing-test-suffix",
		}],
		"title": "file-missing-test-suffix",
		"location": {"file": "policy.rego"},
		"level": "error",
	}}
}

test_success_test_in_file_with_test_suffix if {
	ast := regal.parse_module("policy_test.rego", `package policy_test

	test_foo if { false }
	`)
	r := rule.report with input as ast

	r == set()
}

test_success_test_in_file_named_test if {
	ast := regal.parse_module("test.rego", `package test

	test_foo if { false }
	`)
	r := rule.report with input as ast

	r == set()
}
