package regal.rules.{{.Category}}{{.NameTest}}

import data.regal.ast

import data.regal.rules.{{.Category}}{{.Name}} as rule

# Example test, replace with your own
test_rule_named_foo_not_allowed if {
	module := ast.policy("foo := true")

	r := rule.report with input as module

	# Use print(r) here to see the report. Great for development!

	r == {{ "{{" }}
		"category": "{{.Category}}",
		"description": "Add description of rule here!",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 3,
			"col": 1,
			"end": {
				"row": 3,
				"col": 12,
			},
			"text": "foo := true",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/{{.Category}}/{{.NameOriginal}}",
		}],
		"title": "{{.NameOriginal}}",
	}}
}
