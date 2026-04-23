package regal.rules.bugs["unassigned-return-value_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.rules.bugs["unassigned-return-value"] as rule

test_fail_unused_return_value if {
	r := rule.report with input as ast.with_rego_v1(`allow if {
		indexof("s", "s")
	}`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "bugs",
		"description": "Non-boolean return value unassigned",
		"level": "error",
		"location": {
			"col": 3,
			"row": 6,
			"end": {
				"col": 10,
				"row": 6,
			},
			"file": "policy.rego",
			"text": "\t\tindexof(\"s\", \"s\")",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/unassigned-return-value",
		}],
		"title": "unassigned-return-value",
	}}
}

test_fail_unused_return_value_nested if {
	r := rule.report with input as ast.with_rego_v1(`allow if {
		comprehension := [x |
			indexof("s", "s")
			x := 1
		]
	}`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "bugs",
		"description": "Non-boolean return value unassigned",
		"level": "error",
		"location": {
			"col": 4,
			"end": {
				"col": 11,
				"row": 7,
			},
			"file": "policy.rego",
			"row": 7,
			"text": "\t\t\tindexof(\"s\", \"s\")",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/unassigned-return-value",
		}],
		"title": "unassigned-return-value",
	}}
}

test_success_unused_boolean_return_value if {
	r := rule.report
		with input as ast.policy(`allow if { startswith("s", "s") }`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_success_return_value_assigned if {
	r := rule.report
		with input as ast.policy(`allow if { x := indexof("s", "s") }`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_success_function_arg_return_ignored if {
	r := rule.report
		with config.capabilities as capabilities.provided
		with input as ast.policy(`allow if indexof("s", "s", i)`)

	r == set()
}

test_success_not_triggered_by_print if {
	r := rule.report
		with config.capabilities as capabilities.provided
		with input as ast.policy(`allow if print(lower("A"))`)

	r == set()
}
