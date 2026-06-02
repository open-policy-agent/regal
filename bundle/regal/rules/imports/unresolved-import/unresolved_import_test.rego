package regal.rules.imports["unresolved-import_test"]

import data.regal.aggregators
import data.regal.config

import data.regal.rules.imports["unresolved-import"] as rule

test_fail_identifies_unresolved_imports if {
	p1 := `package foo
	import data.bar
	import data.bar.x
	import data.bar.nope
	import data.nope

	x := 1
	`
	p2 := `package bar
	import data.foo
	import data.foo.x

	x := 1
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == {
		with_location({
			"file": "p1.rego",
			"row": 5,
			"col": 2,
			"end": {
				"col": 8,
				"row": 5,
			},
			"text": "\timport data.nope",
		}),
		with_location({
			"file": "p1.rego",
			"row": 4,
			"col": 2,
			"end": {
				"col": 8,
				"row": 4,
			},
			"text": "\timport data.bar.nope",
		}),
	}
}

test_success_no_unresolved_imports if {
	p1 := `package foo
	import data.bar.x

	x := 1
	`
	p2 := `package bar
	import data.foo.x

	x := 1
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == set()
}

test_success_unresolved_imports_are_excepted if {
	p1 := `package foo
	import data.bar.x
	import data.bar.excepted

	x := 1
	`
	p2 := `package bar

	x := 1
	`
	r := rule.aggregate_report
		with input.aggregates_internal as object.union(
			_imports_agg("p1.rego", p1),
			_imports_agg("p2.rego", p2),
		)
		with config.rules as {"imports": {"unresolved-import": {"except-imports": ["data.bar.excepted"]}}}

	r == set()
}

test_success_unresolved_imports_with_wildcards_are_excepted if {
	p1 := `package foo
	import data.bar.x
	import data.bar.excepted

	x := 1
	`
	r := rule.aggregate_report
		with input.aggregates_internal as _imports_agg("p1.rego", p1)
		with config.rules as {"imports": {"unresolved-import": {"except-imports": ["data.bar.*"]}}}

	r == set()
}

test_success_resolved_import_in_middle_of_explicit_paths if {
	p1 := `package foo
	import data.bar.x.y
	`
	p2 := `package bar

	x.y.z := 1
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == set()
}

test_success_map_rule_resolves if {
	p1 := `package foo
	import data.bar.x
	`
	p2 := `package bar

	x[y] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == set()
}

test_success_map_rule_may_resolve_so_allow if {
	p1 := `package foo
	import data.bar.x.y
	`
	p2 := `package bar

	x[y] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == set()
}

test_success_general_ref_head_rule_may_resolve_so_allow if {
	p1 := `package foo
	import data.bar.x.foo.z.bar
	`
	p2 := `package bar

	x[y].z[foo] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`
	r := rule.aggregate_report with input.aggregates_internal as object.union(
		_imports_agg("p1.rego", p1),
		_imports_agg("p2.rego", p2),
	)

	r == set()
}

test_success_custom_rule_not_flagging_regal_import if {
	p1 := `package custom.regal.bar
	import data.regal.ast

	x := 1
	`
	r := rule.aggregate_report with input.aggregates_internal as _imports_agg("p1.rego", p1)

	r == set()
}

with_location(location) := {
	"category": "imports",
	"description": "Unresolved import",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/unresolved-import",
	}],
	"title": "unresolved-import",
}

_lines(policy) := split(policy, "\n")

_imports_agg(name, policy) := agg if {
	# regal ignore: with-outside-test-context
	aggregated := aggregators with input as regal.parse_module(name, policy)
	agg := {name: {"common": {{
		"imports": aggregated.imports,
		"lines": split(policy, "\n"),
		"rule_tree": aggregated.rule_tree,
	}}}}
}
