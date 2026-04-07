package regal.rules.bugs["sprintf-arguments-mismatch_test"]

import data.regal.ast

import data.regal.rules.bugs["sprintf-arguments-mismatch"] as rule

test_fail_too_many_values_in_array if {
	r := rule.report with input as ast.with_rego_v1(`x := sprintf("%s", [1, 2])`)
	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"row": 5,
			"col": 14,
			"end": {
				"col": 26,
				"row": 5,
			},
			"file": "policy.rego",
			"text": "x := sprintf(\"%s\", [1, 2])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

test_fail_too_many_values_in_array_nested if {
	r := rule.report with input as ast.with_rego_v1(`x := [1 |
		y := [s | s := sprintf("%s", [1, 2])]
	]`)

	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"col": 26,
			"end": {
				"col": 38,
				"row": 6,
			},
			"file": "policy.rego",
			"row": 6,
			"text": "\t\ty := [s | s := sprintf(\"%s\", [1, 2])]",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

test_fail_too_few_values_in_array if {
	r := rule.report with input as ast.with_rego_v1(`x := sprintf("%s%v", [1])`)
	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"row": 5,
			"col": 14,
			"end": {
				"col": 25,
				"row": 5,
			},
			"file": "policy.rego",
			"text": `x := sprintf("%s%v", [1])`,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

test_success_same_number_of_values if {
	r := rule.report with input as ast.policy(`x := sprintf("%s%d", [1, 2])`)

	r == set()
}

test_fail_different_number_of_values_with_explicit_index if {
	r := rule.report with input as ast.with_rego_v1(`x := sprintf("%[1]s %[1]s %[2]d", [1, 2, 3])`)
	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"row": 5,
			"col": 14,
			"end": {
				"col": 44,
				"row": 5,
			},
			"file": "policy.rego",
			"text": "x := sprintf(\"%[1]s %[1]s %[2]d\", [1, 2, 3])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

test_fail_first_arg_is_variable_with_nonmatching_pattern if {
	r := rule.report with input as ast.with_rego_v1(`rule if {
		s := "%s%s"
		sprintf(s, ["foo"])
	}`)
	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"col": 11,
			"end": {
				"col": 21,
				"row": 7,
			},
			"file": "policy.rego",
			"row": 7,
			"text": "\t\tsprintf(s, [\"foo\"])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

test_success_first_arg_is_variable_with_matching_pattern if {
	r := rule.report with input as ast.policy(`rule if {
		s := "%s"
		sprintf(s, ["foo"]) == "foo"
	}`)

	r == set()
}

test_success_same_number_of_values_with_explicit_index if {
	r := rule.report with input as ast.policy(`x := sprintf("%[1]s %[1]s %[2]d", [1, 2])`)

	r == set()
}

test_success_escaped_verbs_are_ignored if {
	r := rule.report with input as ast.policy(`x := sprintf("%d %% %% %s", [1, "f"])`)

	r == set()
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1574
test_fail_padding_not_accounted_for if {
	r := rule.report with input as ast.policy(`x := sprintf("%-*s", ["foo"])`)

	r == {{
		"category": "bugs",
		"description": "Mismatch in `sprintf` arguments count",
		"level": "error",
		"location": {
			"col": 14,
			"end": {
				"col": 29,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "x := sprintf(\"%-*s\", [\"foo\"])",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/sprintf-arguments-mismatch",
		}],
		"title": "sprintf-arguments-mismatch",
	}}
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1574
test_success_padding_accounted_for if {
	r := rule.report with input as ast.policy(`x := sprintf("%-*s", [2, "foo"])`)

	r == set()
}
