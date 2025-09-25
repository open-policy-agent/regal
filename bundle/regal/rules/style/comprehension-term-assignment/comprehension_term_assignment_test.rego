package regal.rules.style["comprehension-term-assignment_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.style["comprehension-term-assignment"] as rule

test_fail_comprehension_term_assignment_last_expr if {
	module := ast.with_rego_v1(`comp := [x |
		some y in input
		x := y
	]`)
	r := rule.report with input as module

	r == {{
		"category": "style",
		"description": "Assigned value can be moved to comprehension term",
		"level": "error",
		"location": {
			"col": 3,
			"end": {
				"col": 7,
				"row": 7,
			},
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tx := y",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/comprehension-term-assignment", "style"),
		}],
		"title": "comprehension-term-assignment",
	}}
}

test_fail_comprehension_term_assignment_not_last_expr if {
	module := ast.with_rego_v1(`comp := [x |
		some y in input
		x := y
		x == 1
	]`)
	r := rule.report with input as module

	r == {{
		"category": "style",
		"description": "Assigned value can be moved to comprehension term",
		"level": "error",
		"location": {"col": 3, "end": {"col": 7, "row": 7}, "file": "policy.rego", "row": 7, "text": "\t\tx := y"},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/comprehension-term-assignment", "style"),
		}],
		"title": "comprehension-term-assignment",
	}}
}

test_fail_comprehension_term_assignment_static_ref if {
	module := ast.with_rego_v1(`comp := [x |
		some y in input
		x := y.attribute
	]`)
	r := rule.report with input as module

	r == {{
		"category": "style",
		"description": "Assigned value can be moved to comprehension term",
		"level": "error",
		"location": {
			"col": 3,
			"end": {"col": 7, "row": 7},
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tx := y.attribute",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/comprehension-term-assignment", "style"),
		}],
		"title": "comprehension-term-assignment",
	}}
}

test_fail_object_comprehension_key_assignment_static_ref if {
	module := ast.with_rego_v1(`comp := {k: v |
		some y, v in input
		k := y.attribute
	}`)

	r := rule.report with input as module
	r == {{
		"category": "style",
		"description": "Assigned value can be moved to comprehension term",
		"level": "error",
		"location": {
			"col": 3,
			"end": {"col": 7, "row": 7},
			"file": "policy.rego",
			"row": 7, "text": "\t\tk := y.attribute",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/comprehension-term-assignment", "style"),
		}],
		"title": "comprehension-term-assignment",
	}}
}

test_fail_object_comprehension_value_assignment_static_ref if {
	module := ast.with_rego_v1(`comp := {k: v |
		some k, y in input
		v := y.attribute
	}`)

	r := rule.report with input as module
	r == {{
		"category": "style",
		"description": "Assigned value can be moved to comprehension term",
		"level": "error",
		"location": {
			"col": 3,
			"end": {"col": 7, "row": 7},
			"file": "policy.rego",
			"row": 7, "text": "\t\tv := y.attribute",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/comprehension-term-assignment", "style"),
		}],
		"title": "comprehension-term-assignment",
	}}
}

test_success_not_flagging_function_call if {
	r := rule.report with input as ast.policy(`comp := [x |
		some y in input
		x := http.send({"method": "get", "url": sprintf("https://example.org/%s", [y])})
	]`)

	r == set()
}

test_success_not_flagging_composite_values if {
	r := rule.report with input as ast.policy(`comp := [x |
		some y in input
		x := {
			"foo": "bar",
			"baz": y,
		}
	]`)

	r == set()
}

test_success_not_flagging_single_expression if {
	r := rule.report with input as ast.policy(`comp := [x | x := input.foo[_].bar]`)

	r == set()
}

test_success_not_flagging_dynamic_ref if {
	r := rule.report with input as ast.policy(`f(x) := [1, x, 3]

	find_vars(node) := [x |
		some var in node
		x := f(var)[_]
	]`)

	r == set()
}

test_success_not_flagging_custom_function_call if {
	r := rule.report with input as ast.policy(`rows := [row |
		some comment in comments
		row := util.to_location_object(comment.location).row
	]`)

	r == set()
}

test_success_not_flagging_assigned_comprehension if {
	r := rule.report with input as ast.policy(`comp := [x |
		some var in input
		x := [y | some y in var]
	]`)

	r == set()
}
