package regal.rules.imports["unresolved-reference_test"]

import data.regal.config
import data.regal.util

import data.regal.rules.imports["unresolved-reference"] as rule

test_fail_identifies_unresolved_reference if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar

	x := bar.baz
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x := 1
	`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-reference",
		[agg1, agg2],
	)

	r == {with_location({
		"file": "p1.rego",
		"row": 4,
		"col": 7,
		"end": {
			"row": 4,
			"col": 14,
		},
		"text": "\tx := bar.baz",
	})}
}

test_success_no_unresolved_reference if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar

	x := bar.baz
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	baz := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}
	r == set()
}

test_fail_identifies_unresolved_reference_with_alias if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar as baz

	x := baz.qux
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x := 1
	`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-reference",
		[agg1, agg2],
	)

	r == {with_location({
		"file": "p1.rego",
		"row": 4,
		"col": 7,
		"end": {
			"row": 4,
			"col": 14,
		},
		"text": "\tx := baz.qux",
	})}
}

test_success_identifies_reference_with_alias if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar as baz

	x := baz.qux
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	qux := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_fail_identifies_unresolved_full_path if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	x := data.bar.baz
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x := 1
	`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-reference",
		[agg1, agg2],
	)

	r == {with_location({
		"file": "p1.rego",
		"row": 3,
		"col": 7,
		"end": {
			"row": 3,
			"col": 19,
		},
		"text": "\tx := data.bar.baz",
	})}
}

test_success_identifies_full_path if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	x := data.bar.baz
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	baz := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_fail_everything_all_at_once if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar
	import data.baz as qux

	x := bar.unknown
	y := qux.unknown
	z := data.qux.unknown
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	known := 1
	`)
	agg3 := rule.aggregate with input as regal.parse_module("p3.rego", `package baz

	known := 1
	`)
	agg4 := rule.aggregate with input as regal.parse_module("p4.rego", `package qux

	known := 1
	`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-reference",
		[agg1, agg2, agg3, agg4],
	)

	r == {
		with_location({
			"file": "p1.rego",
			"row": 5,
			"col": 7,
			"end": {
				"row": 5,
				"col": 18,
			},
			"text": "\tx := bar.unknown",
		}),
		with_location({
			"file": "p1.rego",
			"row": 6,
			"col": 7,
			"end": {
				"row": 6,
				"col": 18,
			},
			"text": "\ty := qux.unknown",
		}),
		with_location({
			"file": "p1.rego",
			"row": 7,
			"col": 7,
			"end": {
				"row": 7,
				"col": 23,
			},
			"text": "\tz := data.qux.unknown",
		}),
	}
}

test_success_everything_all_at_once if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar
	import data.baz as qux

	x := bar.known
	y := qux.known
	z := data.qux.known
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	known := 1
	`)
	agg3 := rule.aggregate with input as regal.parse_module("p3.rego", `package baz

	known := 1
	`)
	agg4 := rule.aggregate with input as regal.parse_module("p4.rego", `package qux

	known := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (((agg1 | agg2) | agg3) | agg4)}
	r == set()
}

test_success_imports_shadowed_by_args_ignored if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	x := 1
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar
	import data.foo

	fun(foo) := foo.bar
	`)

	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}
	r == set()
}

test_success_imports_by_rules_ignored if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo

	x := 1
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar
	import data.foo

	fun := foo.bar if {
		foo := {"bar": 1}
	}
	`)

	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}
	r == set()
}

test_success_builtin_names_are_ignored if {
	agg := rule.aggregate with input as regal.parse_module("p1.rego", `package bar
	import data.time

	fun(foo) := time.now_ns
	`)
		with data.regal.ast.builtin_names as {"time.now_ns"}
		with data.regal.ast.builtin_namespaces as {"time"}

	r := rule.aggregate_report with input as {"aggregate": agg}

	r == set()
}

test_fail_builtin_namespaces_are_not_ignored if {
	agg := rule.aggregate with input as regal.parse_module("p1.rego", `package bar
	import data.time

	fun(foo) := time.now_nss
	`)
		with data.regal.ast.builtin_names as {"time.now_ns"}
		with data.regal.ast.builtin_namespaces as {"time"}

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-reference",
		[agg],
	)

	expected := {with_location({
		"file": "p1.rego",
		"row": 4,
		"col": 14,
		"end": {
			"row": 4,
			"col": 26,
		},
		"text": "\tfun(foo) := time.now_nss",
	})}
	r == expected
}

test_success_ignored_paths_are_ignored if {
	agg := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.datavalues

	x := datavalues.foo
	`)
	r := rule.aggregate_report with input as {"aggregate": agg}
		with config.rules as {"imports": {"unresolved-reference": {"except-paths": ["data.datavalues.*"]}}}

	r == set()
}

with_location(location) := {
	"category": "imports",
	"description": "Unresolved Reference",
	"level": "error",
	"location": location,
	# "location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/unresolved-reference", "imports"),
	}],
	"title": "unresolved-reference",
}
