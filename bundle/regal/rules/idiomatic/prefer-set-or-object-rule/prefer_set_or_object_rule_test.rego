package regal.rules.idiomatic["prefer-set-or-object-rule_test"]

import data.regal.ast

import data.regal.rules.idiomatic["prefer-set-or-object-rule"] as rule

test_fail_set_comprehension_could_be_rule if {
	module := ast.with_rego_v1(`my_set := {s |
		some s in input
		s > 10
	}`)
	r := rule.report with input as module

	r == {{
		"category": "idiomatic",
		"description": "Prefer set or object rule over comprehension",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 3,
				"row": 8,
			},
			"text": "my_set := {s |",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-set-or-object-rule",
		}],
		"title": "prefer-set-or-object-rule",
	}}
}

test_fail_object_comprehension_could_be_rule if {
	module := ast.with_rego_v1(`my_obj := {k: v |
		some k, v in input
		v == "foo"
	}`)
	r := rule.report with input as module

	r == {{
		"category": "idiomatic",
		"description": "Prefer set or object rule over comprehension",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 3,
				"row": 8,
			},
			"text": "my_obj := {k: v |",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-set-or-object-rule",
		}],
		"title": "prefer-set-or-object-rule",
	}}
}

test_success_set_comprehension_array_to_set_conversion_ref_iteration if {
	r := rule.report with input as ast.policy(`my_set := {s | s := arr[_]}`)

	r == set()
}

test_success_set_comprehension_array_to_set_conversion_ref_nested_iteration if {
	r := rule.report with input as ast.policy(`my_set := {s | s := a.b.c[_]}`)

	r == set()
}

test_success_set_comprehension_array_to_set_conversion_ref_nested_iteration_sub_attribute if {
	r := rule.report with input as ast.policy(`my_set := {s | s := a.b.c[_].d}`)

	r == set()
}

test_success_set_comprehension_array_to_set_conversion_some_in if {
	r := rule.report with input as ast.policy(`my_set := {s | some s in arr}`)

	r == set()
}

test_success_set_comprehension_but_rule_body if {
	r := rule.report with input as ast.policy(`my_set := {s | some s in arr; s == ""} if { some_condition }`)

	r == set()
}
