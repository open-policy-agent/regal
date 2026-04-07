package regal.rules.imports["redundant-data-import_test"]

import data.regal.ast

import data.regal.rules.imports["redundant-data-import"] as rule

test_fail_import_data if {
	r := rule.report with input as ast.policy(`import data`)

	r == {{
		"category": "imports",
		"description": "Redundant import of data",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/redundant-data-import",
		}],
		"title": "redundant-data-import",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"text": `import data`,
		},
		"level": "error",
	}}
}

test_fail_import_data_aliased if {
	r := rule.report with input as ast.policy(`import data as d`)

	r == {{
		"category": "imports",
		"description": "Redundant import of data",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/redundant-data-import",
		}],
		"title": "redundant-data-import",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"text": `import data as d`,
		},
		"level": "error",
	}}
}

test_success_import_data_path if {
	r := rule.report with input as ast.policy(`import data.something`)

	r == set()
}
