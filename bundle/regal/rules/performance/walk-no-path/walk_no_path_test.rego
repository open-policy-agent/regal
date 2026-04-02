package regal.rules.performance["walk-no-path_test"]

import data.regal.ast

import data.regal.rules.performance["walk-no-path"] as rule

test_fail_walk_can_have_path_omitted if {
	r := rule.report with input as ast.policy(`
	allow if {
		walk(input, [path, value])
		value == 1
	}
	`)

	r == {with_location({
		"col": 3,
		"end": {
			"col": 7,
			"row": 5,
		},
		"file": "policy.rego",
		"row": 5,
		"text": "\t\twalk(input, [path, value])",
	})}
}

test_success_path_is_wildcard if {
	r := rule.report with input as ast.policy(`
	allow if {
		walk(input, [_, value])
		value == 1
	}
	`)

	r == set()
}

test_success_path_in_head if {
	r := rule.report with input as ast.policy(`
	rule[path] := "foo" if {
		walk(input, [path, value])
		value == 1
	}
	`)

	r == set()
}

test_success_path_in_call if {
	r := rule.report with input as ast.policy(`
	allow if {
		walk(input, [path, value])
		path == 1
	}
	`)

	r == set()
}

test_success_path_in_ref if {
	r := rule.report with input as ast.policy(`
	allow if {
		walk(input, [path, value])
		input[path] == 1
	}
	`)

	r == set()
}

test_success_path_in_ref_in_ref if {
	r := rule.report with input as ast.policy(`
	allow if {
		walk(input, [path, value])
		input[path[0]] == 1
	}
	`)

	r == set()
}

with_location(location) := {
	"category": "performance",
	"description": "Call to `walk` can be optimized",
	"level": "error",
	"location": location,
	"related_resources": [{
		"description": "documentation",
		"ref": "https://www.openpolicyagent.org/projects/regal/rules/performance/walk-no-path",
	}],
	"title": "walk-no-path",
}
