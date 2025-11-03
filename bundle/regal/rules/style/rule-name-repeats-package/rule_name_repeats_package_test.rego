package regal.rules.style["rule-name-repeats-package_test"]

import data.regal.config

import data.regal.rules.style["rule-name-repeats-package"] as rule

related_resources := [{
	"description": "documentation",
	"ref": config.docs.resolve_url("$baseUrl/$category/rule-name-repeats-package", "style"),
}]

base_result := {
	"title": "rule-name-repeats-package",
	"description": "Rule name repeats package",
	"category": "style", "level": "error",
	"related_resources": related_resources,
}

test_rule_empty_if_no_repetition if {
	module := regal.parse_module("example.rego", `
    package policy.foo.bar

    allow := true
    `)

	r := rule.report with input as module

	r == set()
}

test_rule_violation_if_repetition if {
	r := rule.report with input as regal.parse_module("example.rego", `
    package policy.foo.bar

    bar := true
    `)

	r == {object.union(
		base_result,
		{"location": {
			"row": 4,
			"col": 5,
			"end": {
				"col": 8,
				"row": 4,
			},
			"file": "example.rego",
			"text": "    bar := true",
		}},
	)}
}

test_rule_violation_if_repetition_of_more_than_one_path_component if {
	r := rule.report with input as regal.parse_module("example.rego", `package policy.foo.bar.baz
    foo_bar_baz := true

    barBaz := 1
    `)

	r == {
		object.union(base_result, {"location": {
			"row": 2,
			"col": 5,
			"end": {"col": 16, "row": 2},
			"file": "example.rego",
			"text": "    foo_bar_baz := true",
		}}),
		object.union(base_result, {"location": {
			"row": 4,
			"col": 5,
			"end": {"col": 11, "row": 4},
			"file": "example.rego",
			"text": "    barBaz := 1",
		}}),
	}
}

test_rule_violation_if_repetition_multiple if {
	r := rule.report with input as regal.parse_module("example.rego", `
    package policy.foo.bar

    bar := true
    barNumber := 3
    barString := "string"
    `)

	r == {
		object.union(base_result, {"location": {
			"col": 5,
			"file": "example.rego",
			"row": 4,
			"end": {"col": 8, "row": 4},
			"text": "    bar := true",
		}}),
		object.union(base_result, {"location": {
			"col": 5,
			"file": "example.rego",
			"row": 5,
			"end": {"col": 14, "row": 5},
			"text": "    barNumber := 3",
		}}),
		object.union(base_result, {"location": {
			"col": 5,
			"file": "example.rego",
			"row": 6,
			"end": {"col": 14, "row": 6},
			"text": "    barString := \"string\"",
		}}),
	}
}

test_rule_violation_if_repetition_in_function if {
	r := rule.report with input as regal.parse_module("example.rego", `
    package policy.foo.bar

    bar(_) := true
    `)

	r == {object.union(
		base_result,
		{"location": {
			"col": 5,
			"file": "example.rego",
			"row": 4,
			"end": {"col": 8, "row": 4},
			"text": "    bar(_) := true",
		}},
	)}
}

test_rule_violation_if_repetition_in_defaults if {
	r := rule.report with input as regal.parse_module("example.rego", `package policy.foo.bar

    default bar(_) := true
    default barNumber := 3
    `)

	r == {
		object.union(base_result, {"location": {
			"col": 13,
			"file": "example.rego",
			"row": 3,
			"end": {"col": 16, "row": 3},
			"text": "    default bar(_) := true",
		}}),
		object.union(base_result, {"location": {
			"col": 13,
			"file": "example.rego",
			"row": 4,
			"end": {"col": 22, "row": 4},
			"text": "    default barNumber := 3",
		}}),
	}
}

test_rule_violation_if_repetition_ref_head_rule if {
	r := rule.report with input as regal.parse_module("example.rego", `
	package policy

	policy.decision contains "nope"
	`)

	r == {object.union(base_result, {"location": {
		"col": 2,
		"file": "example.rego",
		"row": 4,
		"end": {"col": 8, "row": 4},
		"text": "\tpolicy.decision contains \"nope\"",
	}})}
}
