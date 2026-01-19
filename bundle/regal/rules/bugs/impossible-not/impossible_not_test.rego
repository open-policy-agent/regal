package regal.rules.bugs["impossible-not_test"]

import data.regal.config
import data.regal.util

import data.regal.rules.bugs["impossible-not"] as rule

test_fail_multivalue_not_reference_same_package if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package foo

	test_foo if {
		not partial
	}
	`)
	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == expected_with_location({
		"col": 7,
		"file": "p2.rego",
		"row": 4,
		"end": {
			"col": 14,
			"row": 4,
		},
		"text": "not partial",
	})
}

test_fail_multivalue_not_reference_same_package_nested_expression if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package foo

	test_foo if {
		comprehension := [1 | not partial]
	}`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == expected_with_location({
		"col": 29,
		"end": {
			"col": 36,
			"row": 4,
		},
		"file": "p2.rego",
		"row": 4,
		"text": "not partial",
	})
}

test_fail_multivalue_not_reference_different_package_using_direct_reference if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	test_foo if {
		not data.foo.partial
	}
	`)
	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == expected_with_location({
		"col": 7,
		"file": "p2.rego",
		"row": 4,
		"end": {
			"col": 11,
			"row": 4,
		},
		"text": "not data.foo.partial",
	})
}

test_fail_multivalue_not_reference_different_package_using_import if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"

	another contains "bar"
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	import data.foo

	test_foo if {
		not foo.partial
	}
	`)
	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == expected_with_location({
		"col": 7,
		"file": "p2.rego",
		"row": 6,
		"end": {
			"col": 10,
			"row": 6,
		},
		"text": "not foo.partial",
	})
}

test_success_multivalue_not_reference_invalidated_by_local_var if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	import data.foo

	test_foo if {
		foo := input.bar
		not foo.partial
	}
	`)
	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == set()
}

test_success_multivalue_not_reference_invalidated_by_function_argument if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	import data.foo

	my_function(foo) if {
		not foo.partial
	}
	`)
	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"bugs/impossible-not",
		[agg1, agg2],
	)

	r == set()
}

test_success_multivalue_not_reference_in_same_file_not_reported_in_aggregate_report if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	partial contains "foo"

	test_partial if {
		not partial
	}
	`)
	r := rule.aggregate_report with input as {"aggregate": agg1}

	r == set()
}

test_fail_multivalue_not_reference_in_same_file_reported_in_normal_report if {
	module := regal.parse_module("p1.rego", `package foo

	partial contains "foo"

	test_partial if {
		not partial
	}
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 7,
		"file": "p1.rego",
		"end": {
			"col": 14,
			"row": 6,
		},
		"row": 6,
		"text": "not partial",
	})
}

test_success_multivalue_ref_head_rule_not_accounted_for if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	my.partial[rule] contains "foo" if {
		some rule in input
	}

	test_partial if {
		not partial
	}
	`)
	r := rule.aggregate_report with input as {"aggregate": agg1}

	r == set()
}

expected := {
	"category": "bugs",
	"description": "Impossible `not` condition",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/impossible-not", "bugs"),
	}],
	"title": "impossible-not",
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
