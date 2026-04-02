package regal.rules.idiomatic["use-in-operator_test"]

import data.regal.ast

import data.regal.rules.idiomatic["use-in-operator"] as rule

test_fail_use_in_operator_string_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	"admin" == input.user.roles[_]
	}`)

	r == expected_with_location({
		"col": 13,
		"row": 4,
		"end": {
			"col": 32,
			"row": 4,
		},
		"text": "\t\"admin\" == input.user.roles[_]",
	})
}

test_fail_use_in_operator_number_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	1 == input.lucky_numbers[_]
	}`)

	r == expected_with_location({
		"col": 7,
		"row": 4,
		"end": {
			"col": 29,
			"row": 4,
		},
		"text": "\t1 == input.lucky_numbers[_]",
	})
}

test_fail_use_in_operator_array_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	[1] == input.arrays[_]
	}`)

	r == expected_with_location({
		"col": 9,
		"row": 4,
		"end": {
			"col": 24,
			"row": 4,
		},
		"text": "\t[1] == input.arrays[_]",
	})
}

test_fail_use_in_operator_boolean_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	true == input.booleans[_]
	}`)

	r == expected_with_location({
		"col": 10,
		"row": 4,
		"end": {
			"col": 27,
			"row": 4,
		},
		"text": "\ttrue == input.booleans[_]",
	})
}

test_fail_use_in_operator_object_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	{"x": "y"} == input.objects[_]
	}`)

	r == expected_with_location({
		"col": 16,
		"row": 4,
		"end": {
			"col": 32,
			"row": 4,
		},
		"text": "\t{\"x\": \"y\"} == input.objects[_]",
	})
}

test_fail_use_in_operator_null_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	null == input.objects[_]
	}`)

	r == expected_with_location({
		"col": 10,
		"row": 4,
		"text": "\tnull == input.objects[_]",
		"end": {
			"col": 26,
			"row": 4,
		},
	})
}

test_fail_use_in_operator_set_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	{"foo"} == input.objects[_]
	}`)

	r == expected_with_location({
		"col": 13,
		"row": 4,
		"end": {
			"col": 29,
			"row": 4,
		},
		"text": "\t{\"foo\"} == input.objects[_]",
	})
}

test_fail_use_in_operator_var_lhs if {
	r := rule.report with input as ast.policy(`allow if {
	admin == input.user.roles[_]
	}`)
	r == expected_with_location({
		"col": 11,
		"row": 4,
		"end": {
			"col": 30,
			"row": 4,
		},
		"text": "\tadmin == input.user.roles[_]",
	})
}

test_fail_use_in_operator_string_rhs if {
	r := rule.report with input as ast.policy(`allow if {
	input.user.roles[_] == "admin"
	}`)

	r == expected_with_location({
		"col": 2,
		"row": 4,
		"end": {
			"col": 21,
			"row": 4,
		},
		"text": "\tinput.user.roles[_] == \"admin\"",
	})
}

test_fail_use_in_operator_var_rhs if {
	r := rule.report with input as ast.policy(`allow if {
		input.user.roles[_] == admin
	}`)

	r == expected_with_location({
		"col": 3,
		"row": 4,
		"end": {
			"col": 22,
			"row": 4,
		},
		"text": "\t\tinput.user.roles[_] == admin",
	})
}

test_fail_use_in_operator_ref_lhs if {
	r := rule.report with input as ast.policy(`allow if {
		data.roles.admin == input.user.roles[_]
	}`)

	r == expected_with_location({
		"col": 23,
		"row": 4,
		"end": {
			"col": 42,
			"row": 4,
		},
		"text": "\t\tdata.roles.admin == input.user.roles[_]",
	})
}

test_fail_use_in_operator_ref_rhs if {
	r := rule.report with input as ast.policy(`allow if {
		input.user.roles[_] == data.roles.admin
	}`)

	r == expected_with_location({
		"col": 3,
		"row": 4,
		"end": {
			"col": 22,
			"row": 4,
		},
		"text": "\t\tinput.user.roles[_] == data.roles.admin",
	})
}

test_fail_use_in_operator_scalar_eq_operator if {
	r := rule.report with input as ast.policy(`allow if {
		input.user.roles[_] == data.roles.admin
	}`)

	r == expected_with_location({
		"col": 3,
		"row": 4,
		"end": {
			"col": 22,
			"row": 4,
		},
		"text": "\t\tinput.user.roles[_] == data.roles.admin",
	})
}

test_fail_use_in_operator_ref_eq_operator if {
	r := rule.report with input as ast.policy(`allow if {
		input.user.roles[_] = "foo"
	}`)

	r == expected_with_location({
		"col": 3,
		"row": 4,
		"end": {
			"col": 22,
			"row": 4,
		},
		"text": "\t\tinput.user.roles[_] = \"foo\"",
	})
}

test_success_loop_refs_both_sides if {
	r := rule.report with input as ast.policy(`allow if { required_roles[_] == input.user.roles[_] }`)

	r == set()
}

test_success_uses_in_operator if {
	r := rule.report with input as ast.policy(`allow if { "admin" in input.user.roles }`)

	r == set()
}

expected := {
	"category": "idiomatic",
	"description": "Use in to check for membership",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-in-operator",
	}],
	"title": "use-in-operator",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})} if is_object(location)
