package regal.rules.bugs["redundant-existence-check_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.bugs["redundant-existence-check"] as rule

test_fail_redundant_existence_check if {
	r := rule.report with input as ast.with_rego_v1(`
	redundant if {
		input.foo
		startswith(input.foo, "bar")
	}`)

	r == {{
		"category": "bugs",
		"description": "Redundant existence check",
		"level": "error",
		"location": {
			"col": 3,
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tinput.foo",
			"end": {
				"col": 12,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/redundant-existence-check", "bugs"),
		}],
		"title": "redundant-existence-check",
	}}
}

test_fail_redundant_existence_check_subset if {
	r := rule.report with input as ast.with_rego_v1(`
	redundant if {
		input.foo
		startswith(input.foo.bar, "bar")
	}`)

	r == {{
		"category": "bugs",
		"description": "Redundant existence check",
		"level": "error",
		"location": {
			"col": 3,
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tinput.foo",
			"end": {
				"col": 12,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/redundant-existence-check", "bugs"),
		}],
		"title": "redundant-existence-check",
	}}
}

test_success_not_redundant_existence_check if {
	r := rule.report with input as ast.policy(`
	redundant if {
		input.foo
		something_expensive
		startswith(input.foo, "bar")
	}`)
	r == set()
}

test_success_not_redundant_existence_check_with_cancels if {
	r := rule.report with input as ast.policy(`
	not_redundant if {
		rule.foo with input as {}
		rule.foo == 1
	}`)

	r == set()
}

test_fail_redundant_existence_check_head_assignment_of_ref if {
	r := rule.report with input as ast.with_rego_v1(`
	redundant := input.foo if {
		input.foo
	}`)

	r == {{
		"category": "bugs",
		"description": "Redundant existence check",
		"level": "error",
		"location": {
			"col": 3,
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tinput.foo",
			"end": {
				"col": 12,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/redundant-existence-check", "bugs"),
		}],
		"title": "redundant-existence-check",
	}}
}

test_fail_redundant_existence_check_function_arg if {
	r := rule.report with input as ast.with_rego_v1(`
	fun(foo) if {
		foo
	}`)

	r == {{
		"category": "bugs",
		"description": "Redundant existence check",
		"level": "error",
		"location": {
			"col": 3,
			"end": {
				"col": 6,
				"row": 7,
			},
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tfoo",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/redundant-existence-check", "bugs"),
		}],
		"title": "redundant-existence-check",
	}}
}

test_fail_redundant_existence_check_function_arg_reference_after_use if {
	r := rule.report with input as ast.with_rego_v1(`
	fun(foo) if {
		foo.type == "object"
		foo.type
	}`)

	r == {{
		"category": "bugs",
		"description": "Redundant existence check",
		"level": "error",
		"location": {
			"col": 3,
			"end": {
				"col": 11,
				"row": 8,
			},
			"file": "policy.rego",
			"row": 8,
			"text": "\t\tfoo.type",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/redundant-existence-check", "bugs"),
		}],
		"title": "redundant-existence-check",
	}}
}

test_success_not_redundant_existence_check_negated if {
	r := rule.report with input as ast.policy(`
	not_redundant if {
		rule.foo
		not rule.foo.bar == 1
	}`)

	r == set()
}

test_success_not_redundant_existence_check_arg_negated if {
	# ugly, but not an existence check
	r := rule.report with input as ast.policy(`
	fun(foo) if {
		not foo
	}`)

	r == set()
}

test_success_not_redundant_existence_check_head_ref_negated if {
	# even worse, and should be flagged. but not by this rule
	r := rule.report with input as ast.policy(`
	rule := input.foo if {
		not input.foo < 4
	}`)

	r == set()
}
