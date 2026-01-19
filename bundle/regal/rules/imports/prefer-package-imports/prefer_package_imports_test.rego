package regal.rules.imports["prefer-package-imports_test"]

import data.regal.config

import data.regal.rules.imports["prefer-package-imports"] as rule

test_aggregate_collects_imports_with_location if {
	r := rule.aggregate with input as regal.parse_module("p.rego", `
	package a

	import data.b
	import data.c.d
	`)

	r == {{
		"imports": [
			[["b"], "4:2:4:8"],
			[["c", "d"], "5:2:5:8"],
		],
		"package_path": ["a"],
	}}
}

test_fail_aggregate_report_on_imported_rule if {
	r := rule.aggregate_report with input.aggregates_internal as {
		"policy1.rego": {"imports/prefer-package-imports": {{
			"package_path": ["a"],
			"imports": [
				[["b", "c"], "3:1:3:8"], # likely import of rule — should fail
				[["b"], "4:1:4:8"], # import of package, should not fail
				[["c"], "5:1:5:8"], # unresolved import, should not fail
			],
		}}},
		"policy2.rego": {"imports/prefer-package-imports": {{
			"package_path": ["b"],
			"imports": [],
		}}},
	}

	r == {{
		"category": "imports",
		"description": "Prefer importing packages over rules",
		"level": "error",
		"location": {
			"file": "policy1.rego",
			"col": 1,
			"row": 3,
			"end": {
				"col": 8,
				"row": 3,
			},
			"text": "import data.b.c",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/prefer-package-imports", "imports"),
		}],
		"title": "prefer-package-imports",
	}}
}

test_success_aggregate_report_on_import_with_matching_package if {
	r := rule.aggregate_report with input.aggregate as {
		{"aggregate_data": {
			"package_path": ["a"],
			"imports": [[["b"], "3:1:3:8"]],
		}},
		{"aggregate_data": {
			"package_path": ["b"],
			"imports": [],
		}},
	}

	r == set()
}

# unresolved imports should be flagged by an `unresolved-import`
# rule instead. see https://github.com/open-policy-agent/regal/issues/300
test_success_aggregate_report_on_import_with_unresolved_path if {
	r := rule.aggregate_report with input.aggregate as {
		{"aggregate_data": {
			"package_path": ["a"],
			"imports": [[["b"], "3:1:3:8"]],
		}},
		{"aggregate_data": {
			"package_path": ["bar"],
			"imports": [],
		}},
	}

	r == set()
}

test_success_aggregate_report_ignored_import_path if {
	aggregate := {
		{"aggregate_data": {
			"package_path": ["a"],
			"imports": [[["b", "c"], "3:1:3:8"]],
		}},
		{"aggregate_data": {
			"package_path": ["b"],
			"imports": [],
		}},
	}

	r := rule.aggregate_report with input.aggregate as aggregate
		with config.rules as {"imports": {"prefer-package-imports": {
			"level": "error",
			"ignore-import-paths": ["data.b.c"],
		}}}

	r == set()
}

test_aggregate_ignores_imports_of_regal_in_custom_rule if {
	r := rule.aggregate with input as regal.parse_module("p.rego", `
	package custom.regal.rules.foo.bar

	import data.regal.ast

	import data.a.b.c
	`)

	r == {{
		"imports": [[["a", "b", "c"], "6:2:6:8"]],
		"package_path": ["custom", "regal", "rules", "foo", "bar"],
	}}
}
