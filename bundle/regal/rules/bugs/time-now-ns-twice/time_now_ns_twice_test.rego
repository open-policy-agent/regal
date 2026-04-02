package regal.rules.bugs["time-now-ns-twice_test"]

import data.regal.ast

import data.regal.rules.bugs["time-now-ns-twice"] as rule

test_fail_time_now_ns_called_twice if {
	r := rule.report with input as ast.policy(`
	took := then if {
		now := time.now_ns()
		numbers.range(1, 10)
		then := time.now_ns() - now
	}`)

	r == {{
		"category": "bugs",
		"description": "Repeated calls to `time.now_ns`",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 7,
			"col": 11,
			"end": {
				"row": 7,
				"col": 22,
			},
			"text": "\t\tthen := time.now_ns() - now",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/time-now-ns-twice",
		}],
		"title": "time-now-ns-twice",
	}}
}

test_fail_time_now_ns_called_twice_body_and_head if {
	r := rule.report with input as ast.policy(`
	took := time.now_ns() - now if {
		now := time.now_ns()
		numbers.range(1, 10)
	}`)

	r == {{
		"category": "bugs",
		"description": "Repeated calls to `time.now_ns`",
		"level": "error",
		"location": {
			"file": "policy.rego",
			"row": 5,
			"col": 10,
			"end": {
				"row": 5,
				"col": 21,
			},
			"text": "\t\tnow := time.now_ns()",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/time-now-ns-twice",
		}],
		"title": "time-now-ns-twice",
	}}
}

test_success_time_now_ns_called_once if {
	r := rule.report with input as ast.policy(`
	rule if {
		print(time.now_ns())
	}`)

	r == set()
}

test_success_time_now_ns_called_twice_but_different_rule_indices if {
	r := rule.report with input as ast.policy(`
	rule if {
		print(time.now_ns())
	}

	rule if {
		print(time.now_ns())
	}
	`)

	r == set()
}
