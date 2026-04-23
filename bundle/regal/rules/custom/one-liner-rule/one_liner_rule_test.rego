package regal.rules.custom["one-liner-rule_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.custom["one-liner-rule"] as rule

test_fail_could_be_one_liner if {
	module := ast.policy(`

	allow if {
		input.yes
	}
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 5,
		"end": {
			"col": 7,
			"row": 5,
		},
		"text": "\tallow if {",
	})
}

test_fail_could_be_one_liner_all_keywords if {
	module := ast.policy(`

	allow if {
		input.yes
	}
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"file": "policy.rego",
		"row": 5,
		"end": {
			"col": 7,
			"row": 5,
		},
		"text": "\tallow if {",
	})
}

test_fail_could_be_one_liner_allman_style if {
	module := ast.policy(`

	allow if
	{
		input.yes
	}
	`)
	r := rule.report with input as module

	r == expected_with_location({
		"col": 2,
		"row": 5,
		"end": {
			"col": 7,
			"row": 5,
		},
		"text": "\tallow if",
	})
}

test_success_too_long_for_a_one_liner if {
	module := ast.with_rego_v1(`
	rule := "quite a long text up here" if {
		some_really_long_rule_name_in_fact_53_characters_long == another_long_rule_but_only_45_characters_long
	}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_too_long_for_a_one_liner_configured_line_length if {
	module := ast.with_rego_v1(`
	rule if {
		some_really_long_rule_name_in_fact_53_characters_long
	}
	`)
	r := rule.report
		with input as module
		with config.rules as {"custom": {"one-liner-rule": {"max-line-length": 50}}}

	r == set()
}

test_success_no_one_liner_comment_in_rule_body if {
	module := ast.with_rego_v1(`
	no_one_liner if {
		# Surely one equals one
		1 == 1
	}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_no_one_liner_comment_in_rule_body_same_line if {
	module := ast.with_rego_v1(`
	no_one_liner if {
		1 == 1 # Surely one equals one
	}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_no_one_liner_comment_in_rule_body_line_below if {
	module := ast.with_rego_v1(`
	no_one_liner if {
		1 == 1
		# Surely one equals one
	}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_does_not_use_if_v0 if {
	module := ast.with_rego_v0(`
	allow {
		1 == 1
	}
	`)
	r := rule.report with input as module

	r == set()
}

test_success_already_a_one_liner if {
	r := rule.report with input as ast.with_rego_v1(`allow if 1 == 1`)

	r == set()
}

test_has_notice_if_unmet_capability if {
	r := rule.notices with config.capabilities as {}

	r == {{
		"category": "custom",
		"description": "Missing capability for keyword `if`",
		"level": "notice",
		"severity": "warning",
		"title": "one-liner-rule",
	}}
}

# verify fix for https://github.com/open-policy-agent/regal/issues/1527
test_fail_single_expression_spanning_multiple_lines_already_a_one_liner if {
	module := ast.policy(`

	foo := bar if baz in {
		"foo",
		"bar",
	}
	`)
	r := rule.report with input as module

	r == set()
}

expected := {
	"category": "custom",
	"description": "Rule body could be made a one-liner",
	"level": "error",
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/custom/one-liner-rule",
	}],
	"title": "one-liner-rule",
	"location": {"file": "policy.rego"},
}

expected_with_location(location) := {object.union(expected, {"location": location})}
