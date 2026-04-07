package regal.rules.imports["import-shadows-builtin_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.imports["import-shadows-builtin"] as rule

test_fail_import_shadows_builtin_name if {
	r := rule.report with input as ast.policy(`import data.print`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "imports",
		"description": "Import shadows built-in namespace",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 7,
				"row": 3,
			},
			"text": "import data.print",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/import-shadows-builtin",
		}],
		"title": "import-shadows-builtin",
	}}
}

test_fail_import_shadows_builtin_namespace if {
	r := rule.report with input as ast.policy(`import input.foo.http`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "imports",
		"description": "Import shadows built-in namespace",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 7,
				"row": 3,
			},
			"text": "import input.foo.http",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/import-shadows-builtin",
		}],
		"title": "import-shadows-builtin",
	}}
}

test_success_import_does_not_shadows_builtin_name if {
	r := rule.report with input as ast.policy(`import data.users`) with config.capabilities as capabilities.provided

	r == set()
}

test_success_import_shadows_but_alias_does_not if {
	r := rule.report with input as ast.policy(`import data.http as http_attributes`)
		with config.capabilities as capabilities.provided

	r == set()
}
