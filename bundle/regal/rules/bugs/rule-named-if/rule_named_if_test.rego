package regal.rules.bugs["rule-named-if_test"]

import data.regal.ast
import data.regal.capabilities

import data.regal.rules.bugs["rule-named-if"] as rule

test_fail_rule_named_if if {
	r := rule.report with input as ast.with_rego_v0(`
	allow := true if {
        input.foo
    }`)

	r == {{
		"category": "bugs",
		"description": "Rule named \"if\"",
		"level": "error",
		"location": {
			"col": 16,
			"file": "policy_v0.rego",
			"row": 4,
			"end": {
				"col": 18,
				"row": 4,
			},
			"text": "\tallow := true if {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/rule-named-if",
		}],
		"title": "rule-named-if",
	}}
		with input.regal.file.rego_version as "v0"
		with capabilities.is_opa_v1 as false
}
