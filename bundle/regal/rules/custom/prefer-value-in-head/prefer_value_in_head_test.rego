package regal.rules.custom["prefer-value-in-head_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.custom["prefer-value-in-head"] as rule

test_fail_value_could_be_in_head_assign if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := 10
	}`)

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 10,
			"row": 5,
		},
		"text": "\t\tx := 10",
	})
}

test_fail_value_could_be_in_head_assign_composite if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := {"foo": 10}
	}`)

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 19,
			"row": 5,
		},
		"text": "\t\tx := {\"foo\": 10}",
	})
}

test_fail_value_is_in_head_assign if {
	r := rule.report with input as ast.policy(`value := 10 if { input.x }`)

	r == set()
}

test_fail_value_could_be_in_head_eq if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x = 10
	}`)

	r == expected_with_location({
		"col": 7,
		"file": "policy.rego",
		"row": 5,
		"end": {
			"col": 9,
			"row": 5,
		},
		"text": "\t\tx = 10",
	})
}

test_success_value_is_in_head_eq if {
	r := rule.report with input as ast.policy(`value = x if { input.x }`)

	r == set()
}

test_fail_value_could_be_in_head_but_not_a_scalar if {
	module := ast.policy(`value := x if {
		input.x
		x := [i | i := input[_]]
	}`)
	r := rule.report with input as module
		with config.rules as {"custom": {"prefer-value-in-head": {"only-scalars": true}}}

	r == set()
}

test_fail_value_could_be_in_head_templatestring if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := $"{input.y}"
	}`)

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 20,
			"row": 5,
		},
		"text": "\t\tx := $\"{input.y}\"",
	})
}

test_success_only_scalar_no_include_interpolated if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := $"{input.y}"
	}`)
		with config.rules as {"custom": {"prefer-value-in-head": {"only-scalars": true}}}

	r == set()
}

test_fail_value_could_be_in_head_only_scalars_with_include_interpolated if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := $"{input.y}"
	}`)
		with config.rules as {"custom": {"prefer-value-in-head": {"only-scalars": true, "include-interpolated": true}}}

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 20,
			"row": 5,
		},
		"text": "\t\tx := $\"{input.y}\"",
	})
}

test_fail_value_could_be_in_head_and_is_a_scalar if {
	module := ast.policy(`value := x if {
		input.x
		x := 5
	}`)
	r := rule.report with input as module
		with config.rules as {"custom": {"prefer-value-in-head": {"only-scalars": true}}}

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 9,
			"row": 5,
		},
		"text": "\t\tx := 5",
	})
}

test_fail_value_could_be_in_head_multivalue_rule if {
	r := rule.report with input as ast.with_rego_v1(`violations contains violation if {
		input.bad
		violation := "not good!"
	}`)

	r == expected_with_location({
		"col": 16,
		"row": 7,
		"end": {
			"col": 27,
			"row": 7,
		},
		"text": "\t\tviolation := \"not good!\"",
	})
}

test_fail_value_could_be_in_head_object_rule if {
	r := rule.report with input as ast.policy(`foo["bar"] := x if {
		input.foo
		x := "bar"
	}`)

	r == expected_with_location({
		"col": 8,
		"row": 5,
		"end": {
			"col": 13,
			"row": 5,
		},
		"text": "\t\tx := \"bar\"",
	})
}

test_success_var_name_is_in_excepted_set if {
	r := rule.report with input as ast.policy(`value := x if {
		input.x
		x := 10
	}`)
		with config.rules as {"custom": {"prefer-value-in-head": {"except-var-names": ["x"]}}}

	r == set()
}

expected := {
	"category": "custom",
	"description": "Prefer value in rule head",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": config.docs.resolve_url("$baseUrl/$category/prefer-value-in-head", "custom"),
	}],
	"title": "prefer-value-in-head",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})}
