package regal.main_test

import data.regal.config
import data.regal.main
import data.regal.util

test_ignore_rule_config if {
	policy := `package p

	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "ignore"}}}
		with data.internal.prepared.rules_to_run as {}

	count(report) == 0
}

test_ignore_directive_failure if {
	policy := `package p

	# regal ignore:todo-comment
	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}

	count(report) == 1
}

test_ignore_directive_success if {
	policy := `package p

	# regal ignore:prefer-snake-case
	camelCase := "yes"
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}

	count(report) == 0
}

test_ignore_directive_success_same_line if {
	policy := `package p

	camelCase := "yes" # regal ignore:prefer-snake-case
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}

	count(report) == 0
}

test_ignore_directive_success_same_line_trailing_directive if {
	policy := `package p

	camelCase := "yes" # camelCase is nice! # regal ignore:prefer-snake-case
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}

	count(report) == 0
}

test_ignore_directive_success_same_line_todo_comment if {
	policy := `package p

	camelCase := "yes" # TODO! camelCase isn't nice! # regal ignore:todo-comment
	`
	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as {"style": {"todo-comment": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"style": {"todo-comment"}}

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
		with data.internal.prepared.rules_to_run as {"style": {
			"prefer-snake-case",
			"use-assignment-operator",
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
		with data.internal.prepared.rules_to_run as {"style": {
			"prefer-snake-case",
			"use-assignment-operator",
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
		"aggregates_internal": {"p.rego": {"imports/unresolved-import": [{}]}},
		"regal": {"file": {"name": "p.rego"}},
		"ignore_directives": {},
	}
		with config.rules as {"imports": {"unresolved-import": {"level": "error"}}}
		with data.internal.prepared.rules_to_run as {"imports": {"unresolved-import"}}
		with data.regal.rules.imports["unresolved-import"].aggregate_report as {{
			"category": "imports",
			"level": "error",
			"location": {"col": 1, "file": "p.rego", "row": 6, "text": "import data.provider.parameters"},
			"title": "unresolved-import",
		}}

	count(report_without_ignore_directives) == 1

	report_with_ignore_directives := main.aggregate_report with input as {
		"aggregates_internal": {"p.rego": {"imports/unresolved-import": [{}]}},
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
	pat := config.patterns_compiler(cfg.style["prefer-snake-case"].ignore.files)

	report := main.report with input as regal.parse_module("p.rego", policy)
		with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"style": {"prefer-snake-case"}}
		with data.internal.prepared.ignore_patterns.files.style["prefer-snake-case"] as pat

	count(report) == 0
}

test_exclude_files_rule_config_with_path_prefix_relative_name if {
	rules_config := {"testing": {"test": {
		"level": "error",
		"ignore": {"files": ["bar/*"]},
	}}}
	compiled := config.patterns_compiler(rules_config.testing.test.ignore.files)

	rules_to_run := main._rules_to_run with config.rules as rules_config
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with data.internal.prepared.ignore_patterns.files.testing.test as compiled
		with input.regal.file.name as "bar/p.rego"
		with config.path_prefix as "/foo" # ignored as not prefix of input file

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_path_prefix_relative_name if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with input.regal.file.name as "bar/p.rego"
		with config.path_prefix as "/foo" # ignored as not prefix of input file

	rules_to_run == {"testing": {"test"}}
}

test_exclude_files_rule_config_with_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["bar/*"]}}}}
	pat := config.patterns_compiler(cfg.testing.test.ignore.files)

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with data.internal.prepared.ignore_patterns.files.testing.test as pat
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/foo"

	rules_to_run == {}
}

test_exclude_files_rule_config_with_root_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["foo/*"]}}}}
	pat := config.patterns_compiler(cfg.testing.test.ignore.files)

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with data.internal.prepared.ignore_patterns.files.testing.test as pat
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/"

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}
	pat := config.patterns_compiler(cfg.testing.test.ignore.files)

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with data.internal.prepared.ignore_patterns.files.testing.test as pat
		with input.regal.file.name as "/foo/bar/p.rego"
		with config.path_prefix as "/foo"

	rules_to_run == {"testing": {"test"}}
}

test_exclude_files_rule_config_with_uri_and_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["bar/*"]}}}}
	pat := config.patterns_compiler(cfg.testing.test.ignore.files)

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with data.internal.prepared.ignore_patterns.files.testing.test as pat
		with input.regal.file.name as "file:///foo/bar/p.rego"
		with config.path_prefix as "file:///foo"

	rules_to_run == {}
}

test_not_exclude_files_rule_config_with_uri_and_path_prefix if {
	cfg := {"testing": {"test": {"level": "error", "ignore": {"files": ["notmatching/*"]}}}}

	rules_to_run := main._rules_to_run with config.rules as cfg
		with data.internal.prepared.rules_to_run as {"testing": {"test"}}
		with input.regal.file.name as "file:///foo/bar/p.rego"
		with config.path_prefix as "file:///foo"

	rules_to_run == {"testing": {"test"}}
}

test_force_exclude_file_eval_param if {
	policy := `package p

	camelCase := "yes"
	`

	prep := data.regal.prepared.prepare with config.rules as {"style": {"prefer-snake-case": {"level": "error"}}}
		with data.eval.params.ignore_files as ["p.rego"]

	report := main.report with input as regal.parse_module("p.rego", policy)
		with data.internal.prepared as prep

	count(report) == 0
}

test_force_exclude_file_config if {
	policy := `package p

	camelCase := "yes"
	`
	prep := data.regal.prepared.prepare with config.merged_config as {
		"rules": {"style": {"prefer-snake-case": {"level": "error"}}},
		"ignore": {"files": ["p.rego"]},
	}

	report := main.report with input as regal.parse_module("p.rego", policy)
		with data.internal.prepared as prep

	count(report) == 0
}

test_lint_from_stdin_disables_rules_depending_on_filename_creates_notices if {
	policy := `package p

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
		with data.internal.prepared.rules_to_run as {
			"style": {"prefer-snake-case"},
			"testing": {"file-missing-test-suffix"},
			"idiomatic": {"directory-package-mismatch"},
		}

	violation := util.single_set_item(result.report)
	violation.title == "prefer-snake-case"

	{notice.title | some notice in result.lint.notices} == {"file-missing-test-suffix", "directory-package-mismatch"}
}
