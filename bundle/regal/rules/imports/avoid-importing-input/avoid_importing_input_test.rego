package regal.rules.imports["avoid-importing-input_test"]

import data.regal.ast

import data.regal.rules.imports["avoid-importing-input"] as rule

test_fail_import_input if {
	r := rule.report with input as ast.policy(`import input.foo`)

	r == {{
		"category": "imports",
		"description": "Avoid importing input",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/avoid-importing-input",
		}],
		"title": "avoid-importing-input",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"text": `import input.foo`,
		},
		"level": "error",
	}}
}

test_sucess_import_aliased_input if {
	r := rule.report with input as ast.policy(`import input as tfplan`)

	r == set()
}

test_fail_import_input_aliased_attribute if {
	r := rule.report with input as ast.policy(`import input.foo.bar as barbar`)

	r == {{
		"category": "imports",
		"description": "Avoid importing input",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/avoid-importing-input",
		}],
		"title": "avoid-importing-input",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 13,
				"row": 3,
			},
			"text": `import input.foo.bar as barbar`,
		},
		"level": "error",
	}}
}
