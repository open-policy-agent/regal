package regal.rules.style["yoda-condition_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.style["yoda-condition"] as rule

test_fail_yoda_conditions if {
	module := ast.policy(`rule if {
		"foo" == input.bar

		nested := [foo |
			foo := input.baz
			"foo" == foo
		]
	}`)
	r := rule.report with input as module

	r == expected_with_location([
		{"col": 3, "end": {"row": 4, "col": 11}, "file": "policy.rego", "row": 4, "text": "\t\t\"foo\" == input.bar"},
		{"col": 4, "end": {"row": 8, "col": 12}, "file": "policy.rego", "row": 8, "text": "\t\t\t\"foo\" == foo"},
	])
}

test_fail_yoda_conditions_not_equals if {
	module := ast.policy(`rule if {
		"foo" != input.bar

		nested := [foo |
			foo := input.baz
			"foo" != foo
		]
	}`)
	r := rule.report with input as module

	r == expected_with_location([
		{"col": 3, "end": {"col": 11, "row": 4}, "file": "policy.rego", "row": 4, "text": "\t\t\"foo\" != input.bar"},
		{"col": 4, "end": {"col": 12, "row": 8}, "file": "policy.rego", "row": 8, "text": "\t\t\t\"foo\" != foo"},
	])
}

test_fail_yoda_conditions_greater_and_less_than if {
	module := ast.policy(`rule if {
		1 < count(input.bar)
		1 > count(input.bar)
		1 <= count(input.bar)
		1 >= count(input.bar)
	}`)
	r := rule.report with input as module

	r == expected_with_location([
		{"col": 3, "end": {"row": 4, "col": 6}, "file": "policy.rego", "row": 4, "text": "\t\t1 < count(input.bar)"},
		{"col": 3, "end": {"row": 5, "col": 6}, "file": "policy.rego", "row": 5, "text": "\t\t1 > count(input.bar)"},
		{"col": 3, "end": {"row": 6, "col": 7}, "file": "policy.rego", "row": 6, "text": "\t\t1 <= count(input.bar)"},
		{"col": 3, "end": {"row": 7, "col": 7}, "file": "policy.rego", "row": 7, "text": "\t\t1 >= count(input.bar)"},
	])
}

test_success_no_yoda_condition if {
	r := rule.report with input as ast.policy(`rule if input.bar == "foo"`)

	r == set()
}

test_success_constants_on_both_sides if {
	r := rule.report with input as ast.policy(`rule if "foo" == "foo"`)

	r == set()
}

test_success_exclude_ref_with_vars if {
	r := rule.report with input as ast.policy(`rule if "foo" == input.bar[_]`)

	r == set()
}

expected := {
	"category": "style",
	"description": "Yoda condition",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/yoda-condition", "style"),
	}],
	"title": "yoda-condition",
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)

expected_with_location(location) := {object.union(expected, {"location": loc}) |
	some loc in location
} if {
	is_array(location)
}
