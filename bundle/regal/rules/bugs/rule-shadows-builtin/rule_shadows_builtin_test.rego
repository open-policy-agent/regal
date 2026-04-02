package regal.rules.bugs["rule-shadows-builtin_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.bugs["rule-shadows-builtin"] as rule

test_fail_rule_name_shadows_builtin if {
	r := rule.report with input as ast.policy(`or := 1`) with config.capabilities as {"builtins": {"or": {}}}

	r == {{
		"category": "bugs",
		"description": "Rule name shadows built-in",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/rule-shadows-builtin",
		}],
		"title": "rule-shadows-builtin",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 8,
				"row": 3,
			},
			"text": "or := 1",
		},
		"level": "error",
	}}
}

test_fail_rule_name_shadows_builtin_namespace if {
	r := rule.report with input as ast.policy(`http := "yes"`)
		with config.capabilities as {"builtins": {"http.send": {}}}

	r == {{
		"category": "bugs",
		"description": "Rule name shadows built-in",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/rule-shadows-builtin",
		}],
		"title": "rule-shadows-builtin",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 14,
				"row": 3,
			},
			"text": "http := \"yes\"",
		},
		"level": "error",
	}}
}

test_success_rule_name_does_not_shadows_builtin if {
	r := rule.report with input as ast.policy(`foo := 1`)

	r == set()
}
