package regal.rules.bugs["leaked-internal-reference_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.bugs["leaked-internal-reference"] as rule

test_fail_leaked_internal_reference_in_import if {
	r := rule.report with input as ast.policy(`import data.foo._bar`)

	r == expected_with_location({
		"col": 8,
		"row": 3,
		"end": {
			"col": 21,
			"row": 3,
		},
		"text": "import data.foo._bar",
	})
}

test_fail_leaked_internal_reference_in_rule_head if {
	r := rule.report with input as ast.policy(`var := data.foo._bar`)

	r == expected_with_location({
		"col": 8,
		"file": "policy.rego",
		"row": 3,
		"end": {
			"col": 21,
			"row": 3,
		},
		"text": "var := data.foo._bar",
	})
}

test_fail_leaked_internal_reference_in_rule_body if {
	r := rule.report with input as ast.policy(`rule if {
		x := data.foo._bar
	}`)

	r == expected_with_location({
		"col": 8,
		"row": 4,
		"end": {
			"col": 21,
			"row": 4,
		},
		"text": "\t\tx := data.foo._bar",
	})
}

test_fail_leaked_internal_reference_in_nested_comprehension if {
	r := rule.report with input as ast.policy("rule if comp := [x | x := data.foo._bar]")

	r == expected_with_location({
		"col": 27,
		"row": 3,
		"end": {
			"col": 40,
			"row": 3,
		},
		"text": "rule if comp := [x | x := data.foo._bar]",
	})
}

test_ignore_test_file_by_default if {
	r := rule.report
		with input as ast.policy("foo := data.bar._wow")
		with input.regal.file.name as "p_test.rego"

	r == set()
}

test_ignore_test_file_can_be_disabled if {
	r := rule.report
		with input as ast.policy(`foo := data.bar._wow`)
		with input.regal.file.name as "p_test.rego"
		with config.rules as {"bugs": {"leaked-internal-reference": {"include-test-files": true}}}

	r == expected_with_location({
		"file": "p_test.rego",
		"col": 8,
		"row": 3,
		"end": {
			"col": 21,
			"row": 3,
		},
		"text": "foo := data.bar._wow",
	})
}

expected := {
	"category": "bugs",
	"description": "Outside reference to internal rule or function",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/leaked-internal-reference",
	}],
	"title": "leaked-internal-reference",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
