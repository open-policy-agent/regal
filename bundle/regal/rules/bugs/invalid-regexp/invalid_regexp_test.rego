package regal.rules.bugs["invalid-regexp_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.util

import data.regal.rules.bugs["invalid-regexp"] as rule

test_fail_invalid_regexp if {
	r := rule.report
		with input as ast.policy("r := regex.match(`(`, input.string)")
		with config.capabilities as capabilities.provided

	r == {{
		"category": "bugs",
		"description": "Invalid regular expression",
		"level": "error",
		"location": {
			"col": 18,
			"end": {
				"col": 21,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r := regex.match(`(`, input.string)",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/invalid-regexp",
		}],
		"title": "invalid-regexp",
	}}
}

test_fail_invalid_regexp_for_pattern_function[call] if {
	some call in [
		"regex.find_all_string_submatch_n(`(`, input.x, 1)",
		"regex.find_n(`(`, input.x, 1)",
		"regex.globs_match(`(`, `[abc]`)",
		"regex.globs_match(`[abc]`, `(`)",
		"regex.is_valid(`(`)",
		"regex.match(`(`, input.x)",
		"regex.replace(input.x, `(`, `y`)",
		"regex.split(`(`, input.x)",
		"regex.template_match(`(`, input.x, `<`, `>`)",
	]

	r := rule.report
		with input as ast.policy($"r := {call}")
		with config.capabilities as capabilities.provided

	util.single_set_item(r).location.text == $"r := {call}"
}

test_pass_valid_regexp if {
	r := rule.report
		with input as ast.policy("r := regex.match(`[abc]`, input.string)")
		with config.capabilities as capabilities.provided

	r == set()
}
