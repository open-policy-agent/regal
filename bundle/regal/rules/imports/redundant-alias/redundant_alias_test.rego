package regal.rules.imports["redundant-alias_test"]

import data.regal.ast

import data.regal.rules.imports["redundant-alias"] as rule

test_fail_redundant_alias if {
	r := rule.report with input as ast.policy(`import data.foo.bar as bar`)

	r == {{
		"category": "imports",
		"description": "Redundant alias",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/redundant-alias",
		}],
		"title": "redundant-alias",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"text": "import data.foo.bar as bar",
		},
		"level": "error",
	}}
}

test_success_not_redundant_alias if {
	r := rule.report with input as ast.policy(`import data.foo.bar as valid`)

	r == set()
}
