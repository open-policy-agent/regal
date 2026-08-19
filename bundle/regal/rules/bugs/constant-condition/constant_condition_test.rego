package regal.rules.bugs["constant-condition_test"]

import data.regal.ast

import data.regal.rules.bugs["constant-condition"] as rule

test_fail_simple_constant_condition if {
	r := rule.report with input as ast.policy(`allow if {
	1
	}`)
	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 4,
			"text": "\t1",
			"end": {
				"row": 4,
				"col": 3,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_fail_if_template_string_constant_condition if {
	r := rule.report with input as ast.policy(`allow if $"{input.foo == 1}"`)

	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 10,
			"file": "policy.rego",
			"row": 3,
			"text": `allow if $"{input.foo == 1}"`,
			"end": {
				"row": 3,
				"col": 29,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_success_emplate_string_non_constant_condition if {
	r := rule.report with input as ast.policy(`allow if $"{input.foo}" == "bar"`)

	r == set()
}

test_fail_simple_constant_condition_nested if {
	r := rule.report with input as ast.policy(`allow if {
		every x in [1, 2] {
			1
			x == 2
		}
	}`)

	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 4,
			"end": {
				"col": 5,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5, "text": "\t\t\t1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_success_rule_without_body if {
	r := rule.report with input as ast.policy(`allow := true`)
	r == set()
}

test_fail_rule_with_body_looking_generated if {
	r := rule.report with input as ast.policy(`allow if { true }`)
	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"file": "policy.rego",
			"col": 12,
			"row": 3,
			"end": {
				"row": 3,
				"col": 16,
			},
			"text": "allow if { true }",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_fail_operator_constant_condition if {
	r := rule.report with input as ast.policy(`allow if {
	1 == 1
	}`)
	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 4,
			"text": "\t1 == 1",
			"end": {
				"col": 8,
				"row": 4,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_fail_operator_constant_condition_nested if {
	r := rule.report with input as ast.policy(`nested := [1 |
		c := [2 |
			1 == 1
		]
	]`)

	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 4,
			"end": {
				"col": 10,
				"row": 5,
			},
			"file": "policy.rego",
			"row": 5,
			"text": "\t\t\t1 == 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_success_non_constant_condition if {
	r := rule.report with input as ast.policy(`allow if { 1 == input.one }`)
	r == set()
}

test_success_adding_constant_to_set if {
	r := rule.report with input as ast.policy(`rule contains "message"`)
	r == set()
}

test_success_constant_and_or_operand if {
	r := rule.report with input as ast.policy(`import future.keywords.and
	import future.keywords.or

	allow if {
		input.a and 1
		input.b or "str"
		input.c and 1 == 1
	}`)

	r == set()
}

test_fail_constant_condition_in_body_of_and_operand if {
	# a constant condition in an operand body of more than one expression can safely be removed
	r := rule.report with input as ast.policy(`import future.keywords.and

	allow if {
		{
			input.a
			true
		} and input.b
	}`)

	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 4,
			"file": "policy.rego",
			"row": 8,
			"text": "\t\t\ttrue",
			"end": {
				"row": 8,
				"col": 8,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_success_logical_expr_with_non_constant_operand if {
	r := rule.report with input as ast.policy(`import future.keywords.and
	import future.keywords.or

	allow if {
		1 or input.a
		input.b and "str"
		1 == input.c or 2
		{
			x := 1
			x == 1
		} or 2
	}`)

	r == set()
}

test_fail_constant_logical_expr if {
	r := rule.report with input as ast.policy(`import future.keywords.or

	allow if 1 or 2`)

	r == {{
		"category": "bugs",
		"description": "Constant condition",
		"location": {
			"col": 11,
			"file": "policy.rego",
			"row": 5,
			"text": "\tallow if 1 or 2",
			"end": {
				"row": 5,
				"col": 17,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/constant-condition",
		}],
		"title": "constant-condition",
		"level": "error",
	}}
}

test_fail_constant_logical_expr_variations if {
	# only the outermost logical expression of each is reported
	r := rule.report with input as ast.policy(`import future.keywords.and
	import future.keywords.or

	allow if {
		true and false
		1 or 2 or 3
		1 == 1 or "a" != "b"
	}`)

	locations := {[violation.location.row, violation.location.col] | some violation in r}

	locations == {[7, 3], [8, 3], [9, 3]}
	count(r) == 3
}
