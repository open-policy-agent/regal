package regal.rules.style["messy-rule_test"]

import data.regal.ast

import data.regal.rules.style["messy-rule"] as rule

test_success_non_messy_definition if {
	r := rule.report with input as ast.policy(`
	foo if true

	foo if 5 == 1

	bar if false
	`)

	r == set()
}

test_fail_messy_definition if {
	module := ast.policy(`
	foo if true

	bar if false

	foo if 5 == 1
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 8,
		"end": {
			"col": 15,
			"row": 8,
		},
		"text": "\tfoo if 5 == 1",
	})
}

test_fail_messy_default_definition if {
	module := ast.policy(`
	default foo := true

	bar if false

	foo if 5 == 1
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 8,
		"end": {
			"col": 15,
			"row": 8,
		},
		"text": "\tfoo if 5 == 1",
	})
}

test_fail_messy_nested_rule_definition if {
	module := ast.with_rego_v1(`
	base.foo if true

	bar if false

	base.foo if 5 == 1
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 10,
		"end": {
			"col": 20,
			"row": 10,
		},
		"text": "\tbase.foo if 5 == 1",
	})
}

test_success_non_incremental_nested_rule_definition if {
	r := rule.report with input as ast.policy(`
	base.foo if true

	bar if false

	base.bar if 5 == 1
	`)

	r == set()
}

test_success_non_messy_ref_head_rules if {
	r := rule.report with input as ast.policy(`
	keywords[foo.bar] contains "foo"

	keywords[foo] contains "foo"

	keywords[foo.baz] contains "foo"
	`)

	r == set()
}

test_fail_messy_incremental_nested_variable_rule_definitiion if {
	module := ast.with_rego_v1(`
	base[x].foo := 5 if { x := 1 }

	bar if false

	base[x].foo := 1 if { x := 1 }
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 10,
		"end": {
			"col": 32,
			"row": 10,
		},
		"text": "\tbase[x].foo := 1 if { x := 1 }",
	})
}

test_success_messy_test_is_ignored if {
	module := ast.policy(`
	test_foo if 1

	foo := 1

	test_foo if 2
	`)

	r := rule.report with input as module
	r == set()
}

expected := {
	"category": "style",
	"description": "Messy incremental rule",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/messy-rule",
	}],
	"title": "messy-rule",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
