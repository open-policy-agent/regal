package regal.rules.idiomatic["use-object-union-n_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.idiomatic["use-object-union-n"] as rule

test_fail_nested_object_union_call_nested_first_argument if {
	r := rule.report with input as ast.policy(`r if object.union(object.union({"a": 1}, {"b": 2}), {"c": 3})`)
		with config.capabilities as {"builtins": {"object.union": {}, "object.union_n": {}}}

	r == {_violation({"location": {
		"col": 6,
		"end": {
			"col": 18,
			"row": 3,
		},
		"row": 3,
		"text": "r if object.union(object.union({\"a\": 1}, {\"b\": 2}), {\"c\": 3})",
	}})}
}

test_fail_nested_object_union_call_nested_second_argument if {
	r := rule.report with input as ast.policy(`r if object.union({"a": 1}, object.union({"b": 2}, {"c": 3}))`)
		with config.capabilities as {"builtins": {"object.union": {}, "object.union_n": {}}}

	r == {_violation({"location": {
		"col": 6,
		"end": {
			"col": 18,
			"row": 3,
		},
		"row": 3,
		"text": "r if object.union({\"a\": 1}, object.union({\"b\": 2}, {\"c\": 3}))",
	}})}
}

test_fail_nested_object_union_call_nested_both_arguments if {
	r := rule.report with input as ast.policy(`
		r if object.union(object.union({"a": 1}, {"b": 2}), object.union({"c": 3}, {"d": 4}))
	`)
		with config.capabilities as {"builtins": {"object.union": {}, "object.union_n": {}}}

	r == {_violation({"location": {
		"col": 8,
		"end": {
			"col": 20,
			"row": 4,
		},
		"row": 4,
		"text": "\t\tr if object.union(object.union({\"a\": 1}, {\"b\": 2}), object.union({\"c\": 3}, {\"d\": 4}))",
	}})}
}

test_fail_all_object_union_calls if {
	r := rule.report with input as ast.policy(`r if object.union({"a": 1}, {"b": 2})`)
		with config.rules as {"idiomatic": {"use-object-union-n": {"flag-all-union": true}}}
		with config.capabilities as {"builtins": {"object.union": {}, "object.union_n": {}}}

	r == {_violation({
		"description": "Prefer using `object.union_n` over `object.union`",
		"location": {
			"col": 6,
			"end": {
				"col": 18,
				"row": 3,
			},
			"row": 3,
			"text": "r if object.union({\"a\": 1}, {\"b\": 2})",
		},
	})}
}

_violation(overrides) := object.union(
	{
		"category": "idiomatic",
		"description": "Prefer using `object.union_n` over nested `object.union` calls",
		"level": "error",
		"location": {
			"col": 0,
			"end": {
				"col": 0,
				"row": 0,
			},
			"file": "policy.rego",
			"row": 0,
			"text": "",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-object-union-n",
		}],
		"title": "use-object-union-n",
	},
	overrides,
)
