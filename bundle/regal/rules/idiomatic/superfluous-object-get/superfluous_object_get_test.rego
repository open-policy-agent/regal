package regal.rules.idiomatic["superfluous-object-get_test"]

import data.regal.ast

import data.regal.rules.idiomatic["superfluous-object-get"] as rule

test_fail_equality_expr_default_value_unused if {
	r := rule.report with input as ast.policy(`
		violation if {
			object.get(input, "foo", "default") == "bar"
		}
	`)

	r == {{
		"category": "idiomatic",
		"description": "Superfluous `object.get` call",
		"level": "error",
		"location": {
			"col": 4,
			"end": {
				"col": 39,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5,
			"text": "\t\t\tobject.get(input, \"foo\", \"default\") == \"bar\"",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/superfluous-object-get",
		}],
		"title": "superfluous-object-get",
	}}
}

test_fail_equality_expr_default_value_unused_yoda if {
	r := rule.report with input as ast.policy(`
		violation if {
			"bar" == object.get(input, "foo", "default")
		}
	`)

	r == {{
		"category": "idiomatic",
		"description": "Superfluous `object.get` call",
		"level": "error",
		"location": {
			"col": 13,
			"end": {
				"col": 48,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5,
			"text": "\t\t\t\"bar\" == object.get(input, \"foo\", \"default\")",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/superfluous-object-get",
		}],
		"title": "superfluous-object-get",
	}}
}

test_success_equality_expr_default_value_used if {
	r := rule.report with input as ast.policy(`
		success if {
			object.get(input, ["foo", "bar"], "default") == "default"
		}
	`)

	r == set()
}

test_success_equality_expr_default_value_is_var if {
	r := rule.report with input as ast.policy(`
		var := data.some.value

		success if {
			object.get(input, ["foo", "bar"], var) == "default"
		}
	`)

	r == set()
}

test_success_equality_expr_default_value_is_ref if {
	r := rule.report with input as ast.policy(`
		success if {
			object.get(input, ["foo", "bar"], data.some.value) == "default"
		}
	`)

	r == set()
}
