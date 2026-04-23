package regal.rules.custom["naming-convention_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.custom["naming-convention"] as rule

test_fail_package_name_does_not_match_pattern if {
	r := rule.report
		with input as regal.parse_module("policy.rego", "package foo.bar")
		with config.rules as conventions([{"targets": ["package"], "pattern": `^foo\.bar\..+$`}])

	r == {expected(
		`Naming violation: package name "foo.bar" does not match configured convention`,
		{
			"col": 1,
			"file": "policy.rego",
			"row": 1,
			"end": {
				"col": 8,
				"row": 1,
			},
			"text": "package foo.bar",
		},
	)}
}

test_success_package_name_matches_pattern if {
	r := rule.report
		with input as regal.parse_module("policy.rego", "package foo.bar")
		with config.rules as conventions([{"targets": ["package"], "pattern": `^foo\.bar$`}])

	r == set()
}

test_fail_rule_name_does_not_match_pattern if {
	r := rule.report
		with input as ast.policy(`FOO := true`)
		with config.rules as conventions([{"targets": ["rule"], "pattern": "^[a-z]+$"}])

	r == {expected(
		`Naming violation: rule name "FOO" does not match configured convention`,
		{
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"text": "FOO := true",
		},
	)}
}

test_success_rule_name_matches_pattern if {
	r := rule.report
		with input as ast.policy(`foo := true`)
		with config.rules as conventions([{"targets": ["rule"], "pattern": "^[a-z]+$"}])

	r == set()
}

test_fail_function_name_does_not_match_pattern if {
	r := rule.report
		with input as ast.policy(`fooBar(_) := true`)
		with config.rules as conventions([{"targets": ["function"], "pattern": "^[a-z]+$"}])

	r == {expected(
		`Naming violation: function name "fooBar" does not match configured convention`,
		{
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 18,
				"row": 3,
			},
			"text": "fooBar(_) := true",
		},
	)}
}

test_success_function_name_matches_pattern if {
	r := rule.report
		with input as ast.policy(`foo_bar(_) := true`)
		with config.rules as conventions([{"targets": ["function"], "pattern": "^[a-z_]+$"}])

	r == set()
}

test_fail_var_name_does_not_match_pattern if {
	policy := ast.policy(`
	allow if {
		fooBar := true
		fooBar == true
	}
	`)
	r := rule.report
		with input as policy
		with config.rules as conventions([{"targets": ["variable"], "pattern": "^[a-z_]+$"}])

	r == {expected(
		`Naming violation: variable name "fooBar" does not match configured convention`,
		{
			"col": 3,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 9,
				"row": 5,
			},
			"text": "\t\tfooBar := true",
		},
	)}
}

test_success_var_name_matches_pattern if {
	policy := ast.policy(`
	allow if {
		some foo_bar
		input[foo_bar]
		foo_bar == "works"
	}
	`)
	r := rule.report
		with input as policy
		with config.rules as conventions([{"targets": ["variable"], "pattern": "^[a-z_]+$"}])

	r == set()
}

test_fail_multiple_conventions if {
	policy := regal.parse_module("policy.rego", `package foo.bar

	foo := true

	bar if {
		fooBar := true
		fooBar == true
	}
	`)
	r := rule.report
		with input as policy
		with config.rules as conventions([
			{"targets": ["package"], "pattern": `^acmecorp\.[a-z_\.]+$`},
			{"targets": ["rule", "variable"], "pattern": "^bar$|^foo_bar$"},
		])

	r == {
		expected(
			`Naming violation: package name "foo.bar" does not match configured convention`,
			{
				"col": 1,
				"file": "policy.rego",
				"row": 1,
				"end": {
					"col": 8,
					"row": 1,
				},
				"text": "package foo.bar",
			},
		),
		expected(
			`Naming violation: rule name "foo" does not match configured convention`,
			{
				"col": 2,
				"file": "policy.rego",
				"row": 3,
				"end": {
					"col": 13,
					"row": 3,
				},
				"text": "\tfoo := true",
			},
		),
		expected(
			`Naming violation: variable name "fooBar" does not match configured convention`,
			{
				"col": 3,
				"file": "policy.rego",
				"row": 6,
				"end": {
					"col": 9,
					"row": 6,
				},
				"text": "\t\tfooBar := true",
			},
		),
	}
}

test_fail_variable_name_does_not_match_name_in_list if {
	policy := ast.policy(`
	allow if {
		fooBar := true
		fooBar == true
	}
	`)
	r := rule.report
		with input as policy
		with config.rules as conventions([{"targets": ["var"], "names": ["foo_bar"]}])

	r == {expected(
		`Naming violation: variable name "fooBar" does not match configured convention`,
		{
			"col": 3,
			"file": "policy.rego",
			"row": 5,
			"end": {
				"col": 9,
				"row": 5,
			},
			"text": "\t\tfooBar := true",
		},
	)}
}

test_success_variable_name_matches_name_in_list if {
	policy := ast.policy(`
	allow if {
		foo_bar := true
		foo_bar == true
	}
	`)
	r := rule.report
		with input as policy
		with config.rules as conventions([{"targets": ["variable"], "names": ["foo_bar"]}])

	r == set()
}

expected(description, location) := {
	"category": "custom",
	"description": description,
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/custom/naming-convention",
	}],
	"title": "naming-convention",
}

conventions(arr) := {"custom": {"naming-convention": {
	"level": "error",
	"conventions": arr,
}}}
