package regal.rules.bugs["duplicate-rule_test"]

import data.regal.ast

import data.regal.rules.bugs["duplicate-rule"] as rule

test_fail_simple_duplicate_rule if {
	module := ast.with_rego_v1(`
	allow if {
		input.foo
	}

	allow if {
		input.foo
	}
	`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Duplicate rule found at line 10",
		"level": "error",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 6,
			"text": "\tallow if {",
			"end": {
				"col": 3,
				"row": 8,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/duplicate-rule",
		}],
		"title": "duplicate-rule",
	}}
}

test_success_similar_but_not_duplicate_rule if {
	r := rule.report with input as ast.with_rego_v1(`
	allow if input.foo == "bar"

	allow if input.foo == "bar "
	`)

	r == set()
}

test_fail_multiple_duplicate_rules if {
	module := ast.with_rego_v1(`

	# varying whitespace in each just for good measure
	# these should still count as duplicates

	allow if {
      input.foo
	}

	allow if {
		input.foo
	}

	allow if {
		  input.foo
	}
	`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Duplicate rules found at lines 14, 18",
		"level": "error",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 10,
			"text": "\tallow if {",
			"end": {
				"col": 3,
				"row": 12,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/duplicate-rule",
		}],
		"title": "duplicate-rule",
	}}
}
