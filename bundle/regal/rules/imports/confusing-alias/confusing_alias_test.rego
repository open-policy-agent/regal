package regal.rules.imports["confusing-alias_test"]

import data.regal.ast

import data.regal.rules.imports["confusing-alias"] as rule

test_fail_confusing_aliased_import if {
	r := rule.report with input as ast.policy(`
	import data.a
	import data.a as b
	`)

	r == {violation_with_location({
		"col": 2,
		"end": {
			"col": 8,
			"row": 5,
		},
		"file": "policy.rego",
		"row": 5,
		"text": "\timport data.a as b",
	})}
}

test_fail_multiple_confusing_aliased_imports if {
	r := rule.report with input as ast.policy(`
	import data.a as b
	import data.a as c
	`)

	r == {
		violation_with_location({
			"col": 2,
			"end": {
				"col": 8,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\timport data.a as b",
		}),
		violation_with_location({
			"col": 2,
			"end": {
				"col": 8,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5,
			"text": "\timport data.a as c",
		}),
	}
}

test_success_no_confusing_aliased_import if {
	r := rule.report with input as ast.policy(`
	import data.a
	import data.b

	import data.c as d
	import data.e as f
	`)

	r == set()
}

violation_with_location(location) := {
	"category": "imports",
	"description": "Confusing alias of existing import",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/confusing-alias",
	}],
	"title": "confusing-alias",
}
