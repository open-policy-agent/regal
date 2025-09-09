package regal.main_test

import data.regal.config
import data.regal.main
import data.regal.util

test_basic_input_success if {
	report := main.report with input as regal.parse_module("p.rego", `package p`)
	report == set()
}

test_multiple_failures if {
	policy := `package p

	# both camel case and unification operator
	default camelCase = "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy) with config.rules as {"style": {
		"prefer-snake-case": {"level": "error"},
		"use-assignment-operator": {"level": "error"},
	}}

	count(report) == 2
}

test_expect_failure if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}

	count(report) == 1
}

test_ignore_rule_config if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "ignore"}}}

	count(report) == 0
}

test_ignore_directive_failure if {
	policy := `package p

	# regal ignore:todo-comment
	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}

	count(report) == 1
}

test_ignore_directive_success if {
	policy := `package p

	# regal ignore:prefer-snake-case
	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}

	count(report) == 0
}

test_ignore_directive_success_same_line if {
	policy := `package p

	camelCase := "yes" # regal ignore:prefer-snake-case
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}

	count(report) == 0
}

test_ignore_directive_success_same_line_trailing_directive if {
	policy := `package p

	camelCase := "yes" # camelCase is nice! # regal ignore:prefer-snake-case
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}

	count(report) == 0
}

test_ignore_directive_success_same_line_todo_comment if {
	policy := `package p

	camelCase := "yes" # TODO! camelCase isn't nice! # regal ignore:todo-comment
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"todo-comment": {"level": "error"}}}

	count(report) == 0
}

test_ignore_directive_multiple_success if {
	policy := `package p

	# regal ignore:prefer-snake-case,use-assignment-operator
	default camelCase = "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy) with config.rules as {"style": {
		"prefer-snake-case": {"level": "error"},
		"use-assignment-operator": {"level": "error"},
	}}

	count(report) == 0
}

test_ignore_directive_multiple_mixed_success if {
	policy := `package p

	# regal ignore:prefer-snake-case,todo-comment
	default camelCase = "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy) with config.rules as {"style": {
		"prefer-snake-case": {"level": "error"},
		"use-assignment-operator": {"level": "error"},
	}}

	count(report) == 1
}

test_ignore_directive_collected_in_aggregate_rule if {
	module := regal.parse_module("p.rego", `package p

	# regal ignore:unresolved-import
	import data.unresolved
	`)

	lint := main.lint with input as object.union(module, {"regal": {"operations": ["lint"]}})

	lint.ignore_directives == {"p.rego": {4: ["unresolved-import"]}}
}

test_ignore_directive_enforced_in_aggregate_rule if {
	report_without_ignore_directives := main.aggregate_report with input as {
		"aggregates_internal": {"imports/unresolved-import": []},
		"regal": {"file": {"name": "p.rego"}},
		"ignore_directives": {},
	}
		with config.rules as {"imports": {"unresolved-import": {"level": "error"}}}
		with data.regal.rules.imports["unresolved-import"].aggregate_report as {{
			"category": "imports",
			"level": "error",
			"location": {"col": 1, "file": "p.rego", "row": 6, "text": "import data.provider.parameters"},
			"title": "unresolved-import",
		}}

	count(report_without_ignore_directives) == 1

	report_with_ignore_directives := main.aggregate_report with input as {
		"aggregates_internal": {"imports/unresolved-import": []},
		"regal": {"file": {"name": "p.rego"}},
		"ignore_directives": {"p.rego": {"6": ["unresolved-import"]}},
	}
		with config.rules as {"imports": {"unresolved-import": {"level": "error"}}}
		with data.regal.rules.imports["unresolved-import"].aggregate_report as {{
			"category": "imports",
			"level": "error",
			"location": {"col": 1, "file": "p.rego", "row": 6, "text": "import data.provider.parameters"},
			"title": "unresolved-import",
		}}

	count(report_with_ignore_directives) == 0
}

test_exclude_files_rule_config if {
	policy := `package p

	camelCase := "yes"
	`
	cfg := {"style": {"prefer-snake-case": {"level": "error", "ignore": {"files": ["p.rego"]}}}}
	report := main.report with input as regal.parse_module("p.rego", policy) with config.rules as cfg

	count(report) == 0
}

test_exclude_files_rule_config_with_path_prefix_relative_name if {
	rules_to_run := main._rules_to_run with config.rules as {"testing": {"test": {
		"level": "error",
		"ignore": {"files": ["bar/*"]},
	}}}
		with input.regal.file.name as "bar/p.rego"
		with config.path_prefix as "/foo" # ignored as not prefix of input file

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_path_prefix_relative_name if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "bar/p.rego"
		with config.path_prefix as "/foo" # ignored as not prefix of input file

	rules_to_run == {"testing": {"test"}}
}

test_exclude_files_rule_config_with_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["bar/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/foo"

	rules_to_run == {}
}

test_exclude_files_rule_config_with_root_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["foo/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/"

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/foo"

	rules_to_run == {"testing": {"test"}}
}

test_exclude_files_rule_config_with_uri_and_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["bar/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "file:///foo/bar/p.rego"
		with config.path_prefix as "file:///foo"

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_uri_and_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with input.regal.file.name as "file:///foo/bar/p.rego"
		with config.path_prefix as "file:///foo"

	rules_to_run == {"testing": {"test"}}
}

test_force_exclude_file_eval_param if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.eval.params.ignore_files as ["p.rego"]

	count(report) == 0
}

test_force_exclude_file_config if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.merged_config as {
			"rules": {"style": {"prefer-snake-case": {"level": "error"}}},
			"ignore": {"files": ["p.rego"]},
		}

	count(report) == 0
}

test_lint_from_stdin_disables_rules_depending_on_filename_creates_notices if {
	policy := `package p

import rego.v1

camelCase := "yes"

test_camelcase if {
	camelCase == "yes"
}
`

	module := regal.parse_module("p.rego", policy)
	mock_input := object.union(module, {"regal": {"operations": ["lint"]}})

	result := main with input as mock_input
		with input.regal.file.name as "stdin"
		with config.merged_config as {
			"capabilities": {},
			"rules": {
				"style": {"prefer-snake-case": {"level": "error"}},
				"testing": {"file-missing-test-suffix": {"level": "error"}},
				"idiomatic": {"directory-package-mismatch": {"level": "error"}},
			},
		}

	violation := util.single_set_item(result.report)
	violation.title == "prefer-snake-case"

	{notice.title | some notice in result.lint.notices} == {"file-missing-test-suffix", "directory-package-mismatch"}
}

test_main_lint if {
	policy := `package p
	x = 1`

	module := regal.parse_module("p.rego", policy)

	mock_input := object.union(module, {"regal": {"operations": ["lint"]}})

	cfg := {"style": {"use-assignment-operator": {"level": "error"}}}

	result := main.lint with input as mock_input with config.rules as cfg

	result.violations == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"level": "error",
		"location": {
			"col": 4,
			"file": "p.rego",
			"row": 2,
			"end": {
				"col": 5,
				"row": 2,
			},
			"text": "\tx = 1",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
	}}
	result.ignore_directives == {"p.rego": {}}
	result.notices == set()
}

test_rules_to_run_not_excluded if {
	cfg := {"rules": {"testing": {"test": {"level": "error"}}}}

	rules_to_run := main._rules_to_run with config.merged_config as cfg
		with input.regal.file.name as "p.rego"
		with config.excluded_file as false

	rules_to_run == {"testing": {"test"}}
}

test_notices if {
	notice := {
		"category": "idiomatic",
		"description": "here's a notice",
		"level": "notice",
		"title": "testme notice",
		"severity": "none",
	}

	notices := main.lint.notices with main._rules_to_run as {"idiomatic": {"testme"}}
		with data.regal.rules.idiomatic.testme.notices as {notice} # regal ignore:unresolved-reference
		with input.regal.operations as ["lint"]

	notices == {notice}
}

test_main_fail_when_input_not_object if {
	violation := {
		"category": "error",
		"title": "invalid-input",
		"description": "provided input must be a JSON AST",
	}

	report := main.report with input as []
	report == {violation}
}

test_report_custom_rule_failure if {
	report := main.report with data.custom.regal.rules as {"testing": {"testme": {"report": {{"title": "fail!"}}}}}
		with input as {"package": {}, "regal": {"file": {"name": "p.rego"}}}
		with config.excluded_file as false

	report == {{"title": "fail!"}}
}

test_aggregate_bundled_rule if {
	agg := main.aggregate with main._rules_to_run as {"foo": {"bar"}}
		with data.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}

	agg == {"foo/bar": {"baz"}}
}

test_aggregate_custom_rule if {
	agg := main.aggregate with data.custom.regal.rules as {"foo": {"bar": {"aggregate": {"baz"}}}}
		with config.excluded_file as false
		with input.regal.file.name as "p.rego"

	agg == {"foo/bar": {"baz"}}
}

test_aggregate_report_custom_rule if {
	mock_input := {
		"aggregates_internal": {"custom/test": {}},
		"regal": {
			"file": {"name": "p.rego"},
			"operations": ["aggregate"],
		},
	}

	mock_rules := {"custom": {"test": {"aggregate_report": {{
		"category": "custom",
		"title": "test",
	}}}}}

	report := main.aggregate_report with input as mock_input
		with data.custom.regal.rules as mock_rules

	report == {{"category": "custom", "title": "test"}}

	violations := main.lint.aggregate.violations with input as mock_input
		with data.custom.regal.rules as mock_rules

	violations == report
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1592
test_big_number_causes_no_parser_error if {
	policy := `package p

	big := 1e1000
	`
	module = regal.parse_module("p.rego", policy)

	module.rules[0].head.value.value == 1e1000
}
