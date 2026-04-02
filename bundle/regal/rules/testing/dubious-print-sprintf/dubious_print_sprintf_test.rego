package regal.rules.testing["dubious-print-sprintf_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.testing["dubious-print-sprintf"] as rule

test_fail_print_sprintf if {
	module := ast.policy(`y if {
		print(sprintf("name is: %s domain is: %s", [input.name, input.domain]))
	}`)
	r := rule.report with input as module with config.capabilities as capabilities.provided

	r == {{
		"category": "testing",
		"description": "Dubious use of print and sprintf",
		"level": "error",
		"location": {
			"col": 9,
			"file": "policy.rego",
			"row": 4,
			"end": {
				"col": 16,
				"row": 4,
			},
			"text": "\t\tprint(sprintf(\"name is: %s domain is: %s\", [input.name, input.domain]))",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/testing/dubious-print-sprintf",
		}],
		"title": "dubious-print-sprintf",
	}}
}

test_fail_bodies_print_sprintf if {
	module := ast.policy(`y if {
		comprehension := [x |
			x := input[_]
			print(sprintf("x is: %s", [x]))
		]
	}`)
	r := rule.report with input as module with config.capabilities as capabilities.provided

	r == {{
		"category": "testing",
		"description": "Dubious use of print and sprintf",
		"level": "error",
		"location": {
			"col": 10,
			"file": "policy.rego",
			"row": 6,
			"end": {
				"col": 17,
				"row": 6,
			},
			"text": "\t\t\tprint(sprintf(\"x is: %s\", [x]))",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/testing/dubious-print-sprintf",
		}],
		"title": "dubious-print-sprintf",
	}}
}
