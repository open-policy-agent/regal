package regal.main_test

import data.regal.config
import data.regal.main

test_basic_input_success if {
	report := main.report with input as regal.parse_module("p.rego", `package p`)
	report == set()
}

test_multiple_failures if {
	policy := `package p

	# both camel case and unification operator
	default camelCase = "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy) with config.rules as {"style": {
		"prefer-snake-case": {"level": "error"},
		"use-assignment-operator": {"level": "error"},
	}}
		with data.internal.prepared.rules_to_run as {"style": {
			"prefer-snake-case",
			"use-assignment-operator",
		}}

	count(report) == 2
}

test_expect_failure if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}

	count(report) == 1
}

test_main_lint if {
	policy := `package p
	x = 1`

	module := regal.parse_module("p.rego", policy)

	mock_input := object.union(module, {"regal": {"operations": ["lint"]}})

	cfg := {"style": {"use-assignment-operator": {"level": "error"}}}

	result := main.lint with input as mock_input
		with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"style": {"use-assignment-operator"}}

	result.violations == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"level": "error",
		"location": {
			"col": 4,
			"file": "p.rego",
			"row": 2,
			"end": {
				"col": 5,
				"row": 2,
			},
			"text": "\tx = 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
	}}
	result.ignore_directives == {"p.rego": {}}
	result.notices == set()
}

test_rules_to_run_not_excluded if {
	cfg := {"rules": {"testing": {"test": {"level": "error"}}}}

	rules_to_run := main._rules_to_run with config.merged_config as cfg
		with input.regal.file.name as "p.rego"
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with config.excluded_file as false

	rules_to_run == {"testing": {"test"}}
}

test_main_fail_when_input_not_object if {
	violation := {
		"category": "error",
		"title": "invalid-input",
		"description": "provided input must be a JSON AST",
	}

	report := main.report with input as []
	report == {violation}
}

test_report_custom_rule_failure if {
	report := main.report with data.custom.regal.rules as {"testing": {"testme": {"report": {{"title": "fail!"}}}}}
		with input as {"package": {}, "regal": {"file": {"name": "p.rego"}}}
		with config.excluded_file as false

	report == {{"title": "fail!"}}
}

test_aggregate_bundled_rule if {
	prep := data.regal.prepared.prepare with config.rules as {"foo": {"bar": {"level": "error"}}}

	agg := main.aggregate with data.internal.prepared as prep
		with data.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}
		with input.regal.file.name as "p.rego"

	agg == {"p.rego": {"foo/bar": {"baz"}}}
}

test_aggregate_custom_rule if {
	agg := main.aggregate with data.custom.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}
		with config.excluded_file as false
		with input.regal.file.name as "custom.rego"

	agg["custom.rego"]["foo/bar"] == {"baz"}
}

test_aggregate_report_custom_rule if {
	mock_input := {
		"aggregates_internal": {"p.rego": {"custom/test": [{}]}},
		"regal": {
			"file": {"name": "p.rego"},
			"operations": ["aggregate"],
		},
		"ignore_directives": {},
	}

	mock_rules := {"custom": {"test": {"aggregate_report": {{
		"category": "custom",
		"title": "test",
	}}}}}

	report := main.aggregate_report with input as mock_input
		with data.custom.regal.rules as mock_rules

	report == {{"category": "custom", "title": "test"}}

	violations := main.lint.aggregate.violations with input as mock_input
		with data.custom.regal.rules as mock_rules

	violations == report
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1592
test_big_number_causes_no_parser_error if {
	policy := `package p

	big := 1e1000
	`
	module = regal.parse_module("p.rego", policy)

	module.rules[0].head.value.value == 1e1000
}
