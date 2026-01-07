package regal.rules.bugs["unused-output-variable_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.bugs["unused-output-variable"] as rule

test_fail_unused_output_variable if {
	r := rule.report with input as ast.with_rego_v1(`
	fail if {
		input[x]
	}
	`)

	r == {{
		"category": "bugs",
		"description": "Unused output variable",
		"level": "error",
		"location": {"col": 9, "end": {"col": 10, "row": 7}, "file": "policy.rego", "row": 7, "text": "\t\tinput[x]"},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/unused-output-variable", "bugs"),
		}],
		"title": "unused-output-variable",
	}}
}

test_fail_unused_output_variable_some if {
	r := rule.report with input as ast.with_rego_v1(`
	fail if {
		some xx
		input[xx]
	}
	`)

	r == {{
		"category": "bugs",
		"description": "Unused output variable",
		"level": "error",
		"location": {"col": 9, "end": {"col": 11, "row": 8}, "file": "policy.rego", "row": 8, "text": "\t\tinput[xx]"},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/unused-output-variable", "bugs"),
		}],
		"title": "unused-output-variable",
	}}
}

# NOTE(anders): this should be caught! don't have the time to look into this now though,
# so just leaving this test here for someone to come back to later
_test_fail_unused_output_variable_assignment if {
	# regal ignore:with-outside-test-context
	r := rule.report with input as ast.policy(`
	fail if {
		some y
		x := input[y]
		x == 0
	}
	`)

	r == {{
		"category": "bugs",
		"description": "Unused output variable",
		"level": "error",
		"location": {"col": 14, "end": {"col": 15, "row": 6}, "file": "policy.rego", "row": 6, "text": "\t\tx := input[y]"},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/unused-output-variable", "bugs"),
		}],
		"title": "unused-output-variable",
	}}
}

test_success_unused_wildcard if {
	r := rule.report with input as ast.policy("success if input[_]")

	r == set()
}

test_success_not_unused_variable_in_head_value if {
	r := rule.report with input as ast.policy("success := x if input[x]")

	r == set()
}

test_success_not_unused_variable_in_head_term_value if {
	r := rule.report with input as ast.policy("success := {x} if input[x]")

	r == set()
}

test_success_not_unused_variable_in_head_term_key if {
	r := rule.report with input as ast.policy("success contains {x} if input[x]")

	r == set()
}

test_success_not_unused_variable_in_head_key if {
	r := rule.report with input as ast.policy("success contains x if input[x]")

	r == set()
}

test_success_not_unused_output_variable_function_call if {
	r := rule.report with input as ast.policy(`
	success if {
		some x
		input[x]
		regex.match("[x]", x)
	}
	`)

	r == set()
}

test_success_not_unused_output_variable_function_call_arg_term if {
	r := rule.report with input as ast.policy(`
	success if {
		some x
		input[x]
		f({x})
	}
	`)

	r == set()
}

test_success_not_unused_output_variable_other_ref if {
	r := rule.report with input as ast.policy(`
	success if {
		some x
		input[x] == data.foo[x]
	}
	`)

	r == set()
}

test_success_not_unused_output_variable_head_ref if {
	r := rule.report with input as ast.policy(`
	success[x] if {
		some x
		input[x]
	}
	`)

	r == set()
}

test_success_not_output_variable_rule if {
	r := rule.report with input as ast.policy(`
	x := 1

	success := x if input[x]
	`)

	r == set()
}

test_success_not_output_variable_argument if {
	r := rule.report with input as ast.policy("success(x) if input[x]")

	r == set()
}

test_success_not_unused_comprehension_term if {
	r := rule.report with input as ast.with_rego_v1(`success if {x | input[x]}`)

	r == set()
}

test_success_not_unused_comprehension_term_complex if {
	r := rule.report with input as ast.policy(`success if {[x, y] | input[x][y]}`)

	r == set()
}

test_success_not_unused_comprehension_term_key_value if {
	r := rule.report with input as ast.policy(`success if {x: y | input[x][y]}`)

	r == set()
}
