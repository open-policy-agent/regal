package regal.rules.idiomatic["non-raw-regex-pattern_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.idiomatic["non-raw-regex-pattern"] as rule

test_fail_non_raw_rule_head if {
	r := rule.report
		with input as ast.policy(`x := regex.match("[0-9]+", "1")`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "idiomatic",
		"description": "Use raw strings for regex patterns",
		"level": "error",
		"location": {
			"col": 18,
			"file": "policy.rego",
			"row": 3,
			"text": "x := regex.match(\"[0-9]+\", \"1\")",
			"end": {
				"col": 26,
				"row": 3,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/non-raw-regex-pattern",
		}],
		"title": "non-raw-regex-pattern",
	}}
}

test_fail_non_raw_rule_body if {
	r := rule.report with input as ast.policy(`allow if {
		regex.is_valid("[0-9]+")
	}`)
		with config.capabilities as capabilities.provided
	r == {{
		"category": "idiomatic",
		"description": "Use raw strings for regex patterns",
		"level": "error",
		"location": {
			"col": 18,
			"file": "policy.rego",
			"row": 4,
			"text": "\t\tregex.is_valid(\"[0-9]+\")",
			"end": {
				"col": 26,
				"row": 4,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/non-raw-regex-pattern",
		}],
		"title": "non-raw-regex-pattern",
	}}
}

test_fail_pattern_in_second_arg if {
	r := rule.report
		with input as ast.policy(`r := regex.replace("a", "[a]", "b")`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "idiomatic",
		"description": "Use raw strings for regex patterns",
		"level": "error",
		"location": {
			"col": 25,
			"file": "policy.rego",
			"row": 3,
			"text": "r := regex.replace(\"a\", \"[a]\", \"b\")",
			"end": {
				"col": 30,
				"row": 3,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/non-raw-regex-pattern",
		}],
		"title": "non-raw-regex-pattern",
	}}
}

test_success_when_using_raw_string if {
	r := rule.report
		with input as ast.policy("v := regex.is_valid(`[0-9]+`)")
		with config.capabilities as capabilities.provided

	r == set()
}
