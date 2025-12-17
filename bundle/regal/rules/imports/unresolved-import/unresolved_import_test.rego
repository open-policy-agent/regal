package regal.rules.imports["unresolved-import_test"]

import data.regal.config
import data.regal.util

import data.regal.rules.imports["unresolved-import"] as rule

test_fail_identifies_unresolved_imports if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar
	import data.bar.x
	import data.bar.nope
	import data.nope

	x := 1
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar
	import data.foo
	import data.foo.x

	x := 1
	`)

	r := rule.aggregate_report with input.aggregates_internal as util.with_source_files(
		"imports/unresolved-import",
		[agg1, agg2],
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
			"text": "import data.nope",
		}),
		with_location({
			"file": "p1.rego",
			"row": 4,
			"col": 2,
			"end": {
				"col": 8,
				"row": 4,
			},
			"text": "import data.bar.nope",
		}),
	}
}

test_success_no_unresolved_imports if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x

	x := 1
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar
	import data.foo.x

	x := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_success_unresolved_imports_are_excepted if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x
	import data.bar.excepted

	x := 1
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}
		with config.rules as {"imports": {"unresolved-import": {"except-imports": ["data.bar.excepted"]}}}

	r == set()
}

test_success_unresolved_imports_with_wildcards_are_excepted if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x
	import data.bar.excepted

	x := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": agg1}
		with config.rules as {"imports": {"unresolved-import": {"except-imports": ["data.bar.*"]}}}

	r == set()
}

test_success_resolved_import_in_middle_of_explicit_paths if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x.y
	`)
	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x.y.z := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_success_map_rule_resolves if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x[y] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_success_map_rule_may_resolve_so_allow if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x.y
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x[y] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_success_general_ref_head_rule_may_resolve_so_allow if {
	agg1 := rule.aggregate with input as regal.parse_module("p1.rego", `package foo
	import data.bar.x.foo.z.bar
	`)

	agg2 := rule.aggregate with input as regal.parse_module("p2.rego", `package bar

	x[y].z[foo] := z if {
		some y in input.ys
		z := {"foo": y + 1}
	}
	`)
	r := rule.aggregate_report with input as {"aggregate": (agg1 | agg2)}

	r == set()
}

test_success_custom_rule_not_flagging_regal_import if {
	agg := rule.aggregate with input as regal.parse_module("p2.rego", `package custom.regal.bar
	import data.regal.ast

	x := 1
	`)
	r := rule.aggregate_report with input as {"aggregate": agg}

	r == set()
}

with_location(location) := {
	"category": "imports",
	"description": "Unresolved import",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/unresolved-import", "imports"),
	}],
	"title": "unresolved-import",
}
