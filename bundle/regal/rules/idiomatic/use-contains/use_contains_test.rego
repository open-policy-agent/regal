package regal.rules.idiomatic["use-contains_test"]

import data.regal.ast

import data.regal.rules.idiomatic["use-contains"] as rule

test_fail_should_use_contains if {
	r := rule.report with input as ast.with_rego_v0(`
	import future.keywords

	rule[item] {
		some item in input.items
	}`)

	r == {{
		"category": "idiomatic",
		"description": "Use the `contains` keyword",
		"level": "error",
		"location": {
			"col": 2,
			"file": "policy_v0.rego",
			"row": 6,
			"end": {
				"row": 6,
				"col": 12,
			},
			"text": "\trule[item] {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-contains",
		}],
		"title": "use-contains",
	}}
}

test_success_uses_contains if {
	r := rule.report with input as ast.policy("rule contains item if some item in input.items")

	r == set()
}

test_success_object_rule if {
	r := rule.report with input as ast.policy(`rule[foo] := bar if {
		foo := "bar"
		bar := "baz"
	}`)

	r == set()
}

# https://github.com/open-policy-agent/regal/issues/1212
test_success_contains_without_rule_body if {
	r := rule.report with input as ast.policy(`mask contains "foo"`)

	r == set()
}
