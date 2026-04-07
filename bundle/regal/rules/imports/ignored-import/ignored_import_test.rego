package regal.rules.imports["ignored-import_test"]

import data.regal.ast

import data.regal.rules.imports["ignored-import"] as rule

test_fail_ignored_import if {
	module := ast.policy(`
	import data.foo

	bar := data.foo
	`)
	r := rule.report with input as module

	r == {{
		"category": "imports",
		"description": "Reference ignores import of data.foo",
		"level": "error",
		"location": {
			"col": 9,
			"file": "policy.rego",
			"row": 6,
			"end": {
				"col": 17,
				"row": 6,
			},
			"text": "\tbar := data.foo",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/ignored-import",
		}],
		"title": "ignored-import",
	}}
}

test_fail_ignored_most_specific_import if {
	module := ast.policy(`
	import data.foo
	import data.foo.bar

	bar := data.foo.bar
	`)
	r := rule.report with input as module

	r == {{
		"category": "imports",
		"description": "Reference ignores import of data.foo.bar",
		"level": "error",
		"location": {
			"col": 9,
			"file": "policy.rego",
			"row": 7,
			"end": {
				"col": 21,
				"row": 7,
			},
			"text": "\tbar := data.foo.bar",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/ignored-import",
		}],
		"title": "ignored-import",
	}}
}

test_success_import_not_ignored if {
	module := ast.policy(`
	import data.foo.bar

	foo := bar
	baz := bar.baz
	`)

	r := rule.report with input as module
	r == set()
}

# this is covered by the avoid-importing-input rule,
# and `input` is arguably never unused as it's a global variable
test_success_import_input_not_ignored if {
	module := ast.policy(`import input`)

	r := rule.report with input as module
	r == set()
}

# this is covered by the redundant-data-import rule,
# and `data` can never be considered unused in Rego
test_success_import_data_not_ignored if {
	module := ast.policy(`import data`)

	r := rule.report with input as module
	r == set()
}
