package regal.rules.style["default-over-else_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.style["default-over-else"] as rule

test_fail_conditionless_else_simple_rule if {
	module := ast.policy(`
	x := 1 if {
		input.x
	} else := 2 if {
		input.y
	} else := 3
	`)
	r := rule.report with input as module

	r == with_location({
		"col": 4,
		"file": "policy.rego",
		"row": 8,
		"end": {
			"col": 13,
			"row": 8,
		},
		"text": "\t} else := 3",
	})
}

test_fail_conditionless_else_object_assignment if {
	module := ast.policy(`
	x := {"foo": "bar"} if {
		input.x
	} else := {"bar": "foo"}
	`)
	r := rule.report with input as module

	r == with_location({
		"col": 4,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 26,
			"row": 6,
		},
		"text": "\t} else := {\"bar\": \"foo\"}",
	})
}

test_success_conditionless_else_not_constant if {
	module := ast.policy(`
	y := input.y

	x := {"foo": "bar"} if {
		input.x
	} else := {"bar": y}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_conditionless_else_input_ref if {
	module := ast.policy(`
	x := {"foo": "bar"} if {
		input.x
	} else := input.foo
	`)
	r := rule.report with input as module

	r == set()
}

test_success_conditionless_else_custom_function if {
	module := ast.policy(`
	x(y) := y if {
		input.foo
	} else := 1
	`)
	r := rule.report with input as module

	r == set()
}

test_fail_conditionless_else_custom_function_prefer_default_functions if {
	module := ast.policy(`
	x(y) := y if {
		input.foo
	} else := 1
	`)
	r := rule.report with input as module
		with config.rules as {"style": {"default-over-else": {"prefer-default-functions": true}}}

	r == with_location({
		"col": 4,
		"file": "policy.rego",
		"row": 6,
		"end": {
			"col": 13,
			"row": 6,
		},
		"text": "\t} else := 1",
	})
}

test_success_conditionless_else_custom_function_not_constant if {
	module := ast.policy(`
	x(y) := y + 1 if {
		input.foo
	} else := y
	`)
	r := rule.report with input as module
		with config.rules as {"style": {"default-over-else": {"prefer-default-functions": true}}}

	r == set()
}

with_location(location) := {{
	"category": "style",
	"description": "Prefer default assignment over fallback else",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/default-over-else",
	}],
	"title": "default-over-else",
}}
