package regal.rules.custom["narrow-argument_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.custom["narrow-argument"] as rule

test_fail_can_be_narrowed_single_ref if {
	r := rule.report with input as ast.policy(`
		fun(obj) if obj.number == 1
		fun(obj) if obj.number == 2
	`)

	r == {{
		"category": "custom",
		"description": "Argument obj only referenced as obj.number, value passed can be narrowed",
		"level": "error",
		"location": {
			"col": 7,
			"end": {
				"col": 10,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\t\tfun(obj) if obj.number == 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/narrow-argument", "custom"),
		}],
		"title": "narrow-argument",
	}}
}

test_fail_can_be_narrowed_prefixed_ref if {
	r := rule.report with input as ast.policy(`
		fun(obj) if obj.x.y.number == 1
		fun(obj) if obj.x.y.string == "1"
	`)

	r == {{
		"category": "custom",
		"description": "Argument obj always referenced by a common prefix, value passed can be narrowed to obj.x.y",
		"level": "error",
		"location": {
			"col": 7,
			"end": {
				"col": 10,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\t\tfun(obj) if obj.x.y.number == 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/narrow-argument", "custom"),
		}],
		"title": "narrow-argument",
	}}
}

test_fail_can_be_narrowed_prefixed_array_ref if {
	r := rule.report with input as ast.policy(`
		fun(arr) if arr[0].y.number == 1
		fun(arr) if arr[0].y.string == "1"
	`)

	r == {{
		"category": "custom",
		"description": "Argument arr always referenced by a common prefix, value passed can be narrowed to arr[0].y",
		"level": "error",
		"location": {
			"col": 7,
			"end": {
				"col": 10,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\t\tfun(arr) if arr[0].y.number == 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/narrow-argument", "custom"),
		}],
		"title": "narrow-argument",
	}}
}

test_success_can_not_be_narrowed_prefixed_array_ref_arg_is_least_common_denominator if {
	r := rule.report with input as ast.policy(`
		fun(terms) := [terms[1].location, terms[0].location]
	`)

	r == set()
}

test_success_can_not_be_narrowed_arg_is_least_common_denominator if {
	r := rule.report with input as ast.policy(`
		fun(obj) if obj.typ == "string"
		fun(obj) if obj.val == "string"
	`)

	r == set()
}

test_success_nested_or_variable_path_not_narrowed if {
	r := rule.report with input as ast.policy(`foo(lines) := lines[bar - 1]`)

	r == set()
}
