package regal.rules.style["external-reference_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.rules.style["external-reference"] as rule

test_fail_function_references_input if {
	r := rule.report with input as ast.policy(`f(_) if { input.foo }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 11,
		"file": "policy.rego",
		"row": 3,
		"end": {
			"col": 16,
			"row": 3,
		},
		"text": `f(_) if { input.foo }`,
	})
}

test_fail_function_references_data if {
	r := rule.report with input as ast.policy(`f(_) if { data.foo }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 11,
		"file": "policy.rego",
		"row": 3,
		"end": {
			"col": 15,
			"row": 3,
		},
		"text": `f(_) if { data.foo }`,
	})
}

test_fail_function_references_data_in_expr if {
	r := rule.report with input as ast.policy(`f(x) if {
		x == data.foo
	}`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 8,
		"file": "policy.rego",
		"row": 4,
		"end": {
			"col": 12,
			"row": 4,
		},
		"text": "\t\tx == data.foo",
	})
}

test_fail_function_references_rule if {
	r := rule.report with input as ast.policy(`
foo := "bar"

f(x, y) if {
	x == 5
	y == foo
}
	`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 7,
		"file": "policy.rego",
		"row": 8,
		"end": {
			"col": 10,
			"row": 8,
		},
		"text": `	y == foo`,
	})
}

test_fail_external_reference_in_head_assignment if {
	r := rule.report with input as ast.policy(`f(_) := r`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 9,
		"file": "policy.rego",
		"row": 3,
		"end": {
			"col": 10,
			"row": 3,
		},
		"text": "f(_) := r",
	})
}

test_fail_external_reference_in_head_terms if {
	r := rule.report with input as ast.policy(`f(_) := {"r": r}`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == expected_with_location({
		"col": 15,
		"file": "policy.rego",
		"row": 3,
		"end": {
			"col": 16,
			"row": 3,
		},
		"text": "f(_) := {\"r\": r}",
	})
}

test_success_function_references_no_input_or_data if {
	r := rule.report with input as ast.policy(`f(x) if { x == true }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_no_input_or_data_reverse if {
	r := rule.report with input as ast.policy(`f(x) if { true == x }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_only_own_vars if {
	r := rule.report with input as ast.policy(`f(x) if { y := x; y == 10 }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_only_own_vars_nested if {
	r := rule.report with input as ast.policy(`f(x, z) if { y := x; y == [1, 2, z]}`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_only_own_vars_and_wildcard if {
	r := rule.report with input as ast.policy(`f(x, y) if { _ = x + y }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_return_var if {
	r := rule.report with input as ast.policy(`f(x) := y if { y = true }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_return_vars if {
	r := rule.report with input as ast.policy(`f(x) := [x, y] if { x = false; y = true }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_external_function if {
	r := rule.report with input as ast.policy(`f(x) if { data.foo.bar(x) }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_success_function_references_external_function_in_expr if {
	r := rule.report with input as ast.policy(`f(x) := y if { y := data.foo.bar(x) }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

test_external_references_max_allowed_configuration if {
	module := ast.policy(`f(x) if {
		data.x
		data.y
		data.z
		data.a
	}`)

	r1 := rule.report with input as module
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 4}}}

	r1 == set()

	r2 := rule.report with input as module
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 2}}}

	# note that we could, flag only the external references above the threshold, but that's potentially
	# confusing I think, as there's nothing more "illegal" about the last one than the first?
	count(r2) == 4
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1283
test_success_variable_from_nested_arg_term if {
	r := rule.report with input as ast.policy(`f([x]) := to_number(x)`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"external-reference": {"max-allowed": 0}}}

	r == set()
}

expected := {
	"category": "style",
	"description": "External reference in function",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/external-reference",
	}],
	"title": "external-reference",
	"level": "error",
}

# regal ignore:external-reference
expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
