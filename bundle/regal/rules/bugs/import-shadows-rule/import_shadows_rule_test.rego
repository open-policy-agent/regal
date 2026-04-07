package regal.rules.bugs["import-shadows-rule_test"]

import data.regal.ast

import data.regal.rules.bugs["import-shadows-rule"] as rule

test_fail_import_shadows_rule if {
	r := rule.report with input as ast.policy(`
	import data.foo.bar

	bar := 1
	`)

	r == {{
		"category": "bugs",
		"description": "Import shadows rule",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 6,
			"col": 2,
			"end": {
				"row": 6,
				"col": 10,
			},
			"text": "\tbar := 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/import-shadows-rule",
		}],
		"title": "import-shadows-rule",
	}}
}

test_fail_import_alias_shadows_rule if {
	r := rule.report with input as ast.policy(`
	import data.foo as bar

	bar := 1
	`)

	r == {{
		"category": "bugs",
		"description": "Import shadows rule",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 6,
			"col": 2,
			"end": {
				"row": 6,
				"col": 10,
			},
			"text": "\tbar := 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/import-shadows-rule",
		}],
		"title": "import-shadows-rule",
	}}
}

test_success_import_does_not_shadow_rule if {
	r := rule.report with input as ast.policy(`
	import data.foo.bar

	foo := 1
	`)

	r == set()
}

test_success_import_alias_does_not_shadow_rule if {
	r := rule.report with input as ast.policy(`
	import data.foo as bar

	foo := 1
	`)

	r == set()
}
