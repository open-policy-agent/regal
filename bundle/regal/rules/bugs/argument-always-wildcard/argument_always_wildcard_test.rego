package regal.rules.bugs["argument-always-wildcard_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.bugs["argument-always-wildcard"] as rule

test_fail[name] if {
	some name, case in cases

	r := rule.report with input as ast.policy(case.policy)

	r == {{
		"category": "bugs",
		"description": "Argument is always a wildcard",
		"level": "error",
		"location": object.union({"file": "policy.rego"}, case.location),
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/argument-always-wildcard",
		}],
		"title": "argument-always-wildcard",
	}}
}

cases["single function single argument always a wildcard"] := {
	"policy": "f(_) := 1",
	"location": {
		"col": 3,
		"end": {
			"col": 4,
			"row": 3,
		},
		"row": 3,
		"text": "f(_) := 1",
	},
}

cases["single argument always a wildcard"] := {
	"policy": "f(_) := 1\nf(_) := 2",
	"location": {
		"col": 3,
		"end": {
			"col": 4,
			"row": 3,
		},
		"row": 3,
		"text": "f(_) := 1",
	},
}

cases["single argument always a wildcard default function"] := {
	"policy": "default f(_) := 1\nf(_) := 2",
	"location": {
		"col": 11,
		"end": {
			"col": 12,
			"row": 3,
		},
		"row": 3,
		"text": "default f(_) := 1",
	},
}

cases["multiple argument always a wildcard"] := {
	"policy": "f(x, _) := x + 1\nf(x, _) := x + 2",
	"location": {
		"col": 6,
		"end": {
			"col": 7,
			"row": 3,
		},
		"row": 3,
		"text": "f(x, _) := x + 1",
	},
}

test_success_single_function_single_argument_always_a_wildcard_except_function_name if {
	r := rule.report with input as ast.policy(`mock_f(_) := 1`)
		with config.rules as {"bugs": {"argument-always-wildcard": {"except-function-name-pattern": "^mock_"}}}

	r == set()
}

test_success_multiple_argument_not_always_a_wildcard if {
	r := rule.report with input as ast.policy(`
		f(x, _) := x + 1
		f(_, y) := y + 2
	`)

	r == set()
}
