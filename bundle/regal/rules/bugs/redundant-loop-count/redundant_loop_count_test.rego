package regal.rules.bugs["redundant-loop-count_test"]

import data.regal.ast

import data.regal.rules.bugs["redundant-loop-count"] as rule

test_fail_count_before_loop_gt_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) > 0
		some x in input.coll
	}`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Redundant count before loop",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 5,
			"col": 3,
			"end": {
				"row": 5,
				"col": 20,
			},
			"text": "\t\tcount(input.coll) > 0",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/redundant-loop-count",
		}],
		"title": "redundant-loop-count",
	}}
}

test_fail_count_before_loop_neq_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) != 0
		some x in input.coll
	}`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Redundant count before loop",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 5,
			"col": 3,
			"end": {
				"row": 5,
				"col": 20,
			},
			"text": "\t\tcount(input.coll) != 0",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/redundant-loop-count",
		}],
		"title": "redundant-loop-count",
	}}
}

test_fail_count_before_key_value_loop_gt_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) > 0
		some x, y in input.coll
	}`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Redundant count before loop",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 5,
			"col": 3,
			"end": {
				"row": 5,
				"col": 20,
			},
			"text": "\t\tcount(input.coll) > 0",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/redundant-loop-count",
		}],
		"title": "redundant-loop-count",
	}}
}

test_fail_count_before_key_value_loop_neq_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) != 0
		some x, y in input.coll
	}`)
	r := rule.report with input as module

	r == {{
		"category": "bugs",
		"description": "Redundant count before loop",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 5,
			"col": 3,
			"end": {
				"row": 5,
				"col": 20,
			},
			"text": "\t\tcount(input.coll) != 0",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/redundant-loop-count",
		}],
		"title": "redundant-loop-count",
	}}
}

test_success_count_not_gt_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) > 3
		some x in input.coll
	}`)
	r := rule.report with input as module

	r == set()
}

test_success_count_not_neq_zero if {
	module := ast.policy(`
	allow if {
		count(input.coll) != 3
		some x in input.coll
	}`)
	r := rule.report with input as module

	r == set()
}
