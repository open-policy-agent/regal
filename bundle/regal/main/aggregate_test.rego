package regal.main_test

import data.regal.config
import data.regal.main

test_aggregate_bundled_rule if {
	prep := data.regal.prepared.prepare with config.rules as {"foo": {"bar": {"level": "error"}}}

	agg := main.aggregate
		with data.internal.prepared as prep
		with data.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}
		with input.regal.file.name as "p.rego"

	agg == {"p.rego": {"foo/bar": {"baz"}}}
}

test_aggregate_custom_rule if {
	agg := main.aggregate
		with data.custom.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}
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

	report := main.aggregate_report
		with input as mock_input
		with data.custom.regal.rules as mock_rules

	report == {{"category": "custom", "title": "test"}}

	violations := main.lint.aggregate.violations
		with input as mock_input
		with data.custom.regal.rules as mock_rules

	violations == report
}
