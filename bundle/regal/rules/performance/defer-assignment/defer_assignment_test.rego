package regal.rules.performance["defer-assignment_test"]

import data.regal.ast

import data.regal.rules.performance["defer-assignment"] as rule

test_fail_can_defer_assignment_simple if {
	r := rule.report with input as ast.with_rego_v1(`
	allow if {
		resp := http.send({"method": "get", "url": "http://localhost"})
		input.foo == "bar"
		resp.status_code == 200
	}
	`)

	r == {with_location({
		"col": 3,
		"end": {
			"col": 66,
			"row": 7,
		},
		"file": "policy.rego",
		"row": 7,
		"text": "\t\tresp := http.send({\"method\": \"get\", \"url\": \"http://localhost\"})",
	})}
}

# note that this is currently a simplistic model — some of the cases below may actually be
# be deferrable, but we won't e.g. defer assignments into loop bodies, etc

test_success_can_not_defer_assignment_var_used_in_next_expression if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		x == true
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_var_used_in_next_negated_expression if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		not x
	}
	`)
	r == set()
}

test_success_can_not_defer_loop_assignment if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input[foo][bar]
		input.x == 2
	}
	`)

	r == set()
}

test_success_can_not_defer_array_assignment if {
	r := rule.report with input as ast.policy(`
	allow if {
		[x, y] := foo("bar")
		input.x == 2
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_in_group if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := 1
		y := 2
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_var_in_rule_head if {
	r := rule.report with input as ast.policy(`
	rule[x] := 1 if {
		x := input.x
		input.bar == "baz"
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_some if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		some foo in bar
		x == 5
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_every if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		every foo in bar {
			foo == x
		}
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_walk if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		walk(input, [p, v])
		v == x
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_has_with if {
	r := rule.report with input as ast.policy(`
	test_allow if {
		x := input.x
		allow with input as x
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_with_in_arg if {
	r := rule.report with input as ast.policy(`
	rule if {
		u := "u"
		input == "U" with input as upper(u)
	}
	`)

	r == set()
}

test_success_can_not_defer_assignment_next_expression_print_call if {
	r := rule.report with input as ast.policy(`
	allow if {
		x := input.x
		print("here!")
		x == "yes"
	}
	`)

	r == set()
}

with_location(location) := {
	"category": "performance",
	"description": "Assignment can be deferred",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/performance/defer-assignment",
	}],
	"title": "defer-assignment",
}
