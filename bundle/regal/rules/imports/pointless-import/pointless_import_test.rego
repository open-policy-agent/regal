package regal.rules.imports["pointless-import_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.imports["pointless-import"] as rule

test_fail_pointless_import_of_same_package if {
	r := rule.report with input as ast.policy("import data.policy")

	r == {{
		"category": "imports",
		"description": "Importing own package is pointless",
		"level": "error",
		"location": {
			"col": 8,
			"end": {
				"col": 19,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "import data.policy",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/pointless-import", "imports"),
		}],
		"title": "pointless-import",
	}}
}

test_success_external_ref_not_flagged if {
	r := rule.report with input as ast.policy("import data.policy.a.b.c")

	r == set()
}

test_success_ref_defined_in_module_flagged if {
	r := rule.report with input as ast.policy(`import data.policy.a.b.c
	
	a.b.c := 1
	`)

	r == {{
		"category": "imports",
		"description": "Importing own package is pointless",
		"level": "error",
		"location": {
			"col": 8,
			"row": 3,
			"end": {
				"col": 25,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "import data.policy.a.b.c",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/pointless-import",
		}],
		"title": "pointless-import",
	}}
}

test_success_ref_prefix_defined_in_module_flagged if {
	r := rule.report with input as ast.policy(`import data.policy.a.b
	
	a.b.c := 1`)

	r == {{
		"category": "imports",
		"description": "Importing own package is pointless",
		"level": "error",
		"location": {
			"col": 8,
			"row": 3,
			"end": {
				"col": 23,
				"row": 3,
			},
			"file": "policy.rego",
			"text": "import data.policy.a.b",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/pointless-import",
		}],
		"title": "pointless-import",
	}}
}
