package regal.rules.performance["repeated-computation_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.performance["repeated-computation"] as rule

test_fail_repeated_builtin_call_same_scope if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.s) > 0
	limit := count(input.s)
	limit > 1
}
`)
		with config.capabilities as capabilities.provided

	r == with_location({
		"col": 11,
		"end": {
			"col": 24,
			"row": 6,
		},
		"file": "policy.rego",
		"row": 6,
		"text": "\tlimit := count(input.s)",
	})
}

test_ok_single_builtin_call if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.s) > 0
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_ok_different_arguments if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.a) > 0
	count(input.b) > 0
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_ok_comprehension_scope if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.s) > 0
	counts := [count(input.s) | some item in input.items; item]
	count(counts) > 0
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_ok_every_scope if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.s) > 0
	every item in input.items {
		count(input.s) > 0
		item
	}
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_fail_three_repetitions_without_pairwise_reports if {
	r := rule.report with input as ast.policy(`
allow if {
	count(input.s) > 0
	count(input.s) > 1
	count(input.s) > 2
}
`)
		with config.capabilities as capabilities.provided

	count(r) == 2

	some first in r
	first.location.row == 6
	first.location.text == "\tcount(input.s) > 1"

	some second in r
	second.location.row == 7
	second.location.text == "\tcount(input.s) > 2"
}

test_ok_custom_function_calls if {
	r := rule.report with input as ast.policy(`
matches(xs) if {
	count(xs) > 0
}

allow if {
	matches(input.s)
	matches(input.s)
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_ok_local_var_argument if {
	r := rule.report with input as ast.policy(`
allow if {
	xs := input.s
	count(xs) > 0
	count(xs) > 1
}
`)
		with config.capabilities as capabilities.provided

	r == set()
}

test_ok_nondeterministic_builtin if {
	r := rule.report with input as ast.policy(`
allow if {
	time.now_ns() > 0
	time.now_ns() > 1
}
`)
		with config.capabilities as capabilities_with_nondeterministic

	r == set()
}

capabilities_with_nondeterministic := object.union(capabilities.provided, {
	"builtins": object.union(capabilities.provided.builtins, {
		"time.now_ns": object.union(capabilities.provided.builtins["time.now_ns"], {"nondeterministic": true}),
	}),
})

with_location(location) := {{
	"category": "performance",
	"description": "Repeated computation",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/performance/repeated-computation",
	}],
	"title": "repeated-computation",
}}
