package regal.rules.custom["forbidden-function-call_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.custom["forbidden-function-call"] as rule

test_fail_forbidden_function if {
	module := ast.policy(`foo := http.send({"method": "GET", "url": "https://example.com"})`)

	r := rule.report
		with input as module
		with config.capabilities as capabilities.provided
		with config.rules as {"custom": {"forbidden-function-call": {
			"level": "error",
			"forbidden-functions": ["http.send"],
		}}}

	r == {{
		"category": "custom",
		"description": "Forbidden function call",
		"level": "error",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 17,
				"row": 3,
			},
			"text": `foo := http.send({"method": "GET", "url": "https://example.com"})`,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/custom/forbidden-function-call",
		}],
		"title": "forbidden-function-call",
	}}
}
