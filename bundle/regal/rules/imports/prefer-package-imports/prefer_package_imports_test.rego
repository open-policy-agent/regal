package regal.rules.imports["prefer-package-imports_test"]

import data.regal.config

import data.regal.rules.imports["prefer-package-imports"] as rule

test_fail_aggregate_report_on_imported_rule if {
	r := rule.aggregate_report with input.aggregates_internal as {
		"policy1.rego": {
			"common": {{
				"package_path": ["a"],
				"lines": [
					"package a",
					"",
					"import data.b.c",
					"import data.b",
					"import data.c",
				],
				"imports": [
					[["b", "c"], "3:1:3:8"], # likely import of rule — should fail
					[["b"], "4:1:4:8"], # import of package, should not fail
					[["c"], "5:1:5:8"], # unresolved import, should not fail
				],
			}},
		},
		"policy2.rego": {
			"common": {{
				"package_path": ["b"],
				"lines": [
					"package b",
				],
				"imports": [],
			}},
		},
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
				"col": 7,
				"row": 3,
			},
			"text": "import data.b.c",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/prefer-package-imports",
		}],
		"title": "prefer-package-imports",
	}}
}

test_fail_import_of_rule if {
	r := rule.aggregate_report with input.aggregates_internal as {
		"a.rego": {"common": {{
			"package_path": ["a"],
			"lines": [
				"package a",
				"",
				"import data.b.c",
			],
			"imports": [[["b", "c"], "3:1:3:8"]],
		}}},
		"b.rego": {"common": {{
			"package_path": ["b"],
			"lines": [
				"package b",
				"",
				"c := 1",
			],
			"imports": [],
		}}},
	}

	r != set()
}

test_success_import_with_matching_package if {
	r := rule.aggregate_report with input.aggregates_internal as {{
		"a.rego": {"common": {{
			"package_path": ["a"],
			"lines": [
				"package a",
				"",
				"import data.b",
			],
			"imports": [[["b"], "3:1:3:8"]],
		}}},
		"b.rego": {"common": {{
			"package_path": ["b"],
			"lines": ["package b"],
			"imports": [],
		}}},
	}}

	r == set()
}

test_success_aggregate_report_on_import_with_unresolved_path if {
	r := rule.aggregate_report with input.aggregates_internal as {{
		"a.rego": {"common": {{
			"package_path": ["a"],
			"lines": [
				"package a",
				"",
				"import data.b",
			],
			"imports": [[["b"], "3:1:3:8"]],
		}}},
		"b.rego": {"common": {{
			"package_path": ["bar"],
			"lines": ["package bar"],
			"imports": [],
		}}},
	}}

	r == set()
}

test_success_aggregate_report_ignored_import_path if {
	aggregates_internal := {
		"a.rego": {"common": {{
			"package_path": ["a"],
			"lines": [
				"package a",
				"",
				"import data.b.c",
			],
			"imports": [[["b", "c"], "3:1:3:8"]],
		}}},
		"b.rego": {"common": {{
			"package_path": ["b"],
			"lines": ["package b"],
			"imports": [],
		}}},
	}

	r := rule.aggregate_report
		with input.aggregates_internal as aggregates_internal
		with config.rules as {"imports": {"prefer-package-imports": {
			"level": "error",
			"ignore-import-paths": ["data.b.c"],
		}}}

	r == set()
}
