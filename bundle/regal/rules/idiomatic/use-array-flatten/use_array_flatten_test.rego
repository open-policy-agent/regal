package regal.rules.idiomatic["use-array-flatten_test"]

import data.regal.ast
import data.regal.config

import data.regal.rules.idiomatic["use-array-flatten"] as rule

# Note: the location is always reported from the outer call to `array.concat`, not the nested one,
# and this is why the location data (excluding the text) looks identical for both test cases below.
# Nested calls are only reported if they too have another nested call as an argument, making them
# the outer call in that case.

test_fail_nested_array_concat_call_nested_first_argument if {
	r := rule.report with input as ast.policy(`r if array.concat(array.concat([1, 2], [3, 4]), [5, 6])`)
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({"location": {
		"col": 6,
		"end": {
			"col": 18,
			"row": 3,
		},
		"row": 3,
		"text": "r if array.concat(array.concat([1, 2], [3, 4]), [5, 6])",
	}})}
}

test_fail_nested_array_concat_call_nested_second_argument if {
	r := rule.report with input as ast.policy(`r if array.concat([1, 2], array.concat([3, 4], [5, 6]))`)
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({"location": {
		"col": 6,
		"end": {
			"col": 18,
			"row": 3,
		},
		"file": "policy.rego",
		"row": 3,
		"text": "r if array.concat([1, 2], array.concat([3, 4], [5, 6]))",
	}})}
}

test_fail_array_concat_call_nested_both_arguments if {
	r := rule.report with input as ast.policy(`
	r if array.concat(array.concat([1, 2], [3, 4]), array.concat([5, 6], [7, 8]))
	`)
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({"location": {
		"col": 7,
		"end": {
			"col": 19,
			"row": 4,
		},
		"file": "policy.rego",
		"row": 4,
		"text": "\tr if array.concat(array.concat([1, 2], [3, 4]), array.concat([5, 6], [7, 8]))",
	}})}
}

test_fail_array_concat_call_nested_two_levels if {
	r := rule.report with input as ast.policy(`
	r if array.concat(array.concat(array.concat([1], [2]), [3]), [4])
	`)
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {
		_violation({"location": {
			"col": 7,
			"end": {
				"col": 19,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\tr if array.concat(array.concat(array.concat([1], [2]), [3]), [4])",
		}}),
		_violation({"location": {
			"col": 20,
			"end": {
				"col": 32,
				"row": 4,
			},
			"file": "policy.rego",
			"row": 4,
			"text": "\tr if array.concat(array.concat(array.concat([1], [2]), [3]), [4])",
		}}),
	}
}

test_success_not_nested_array_concat_call if {
	r := rule.report with input as ast.policy(`r if array.concat([1, 2], [3, 4])`)
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == set()
}

test_fail_array_concat_call_with_array_literals_wrapping_arguments if {
	r := rule.report with input as ast.policy(`r if array.concat([a], [b])`)
		with config.rules as {"idiomatic": {"use-array-flatten": {"flag-wrapped-concat": true}}}
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({
		"description": "Prefer `array.flatten` over `array.concat` with array literals wrapping arguments",
		"location": {
			"col": 6,
			"end": {
				"col": 18,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r if array.concat([a], [b])",
		},
	})}
}

test_fail_array_concat_call_with_only_first_argument_wrapped_by_array_literal if {
	r := rule.report with input as ast.policy(`r if array.concat([a], b)`)
		with config.rules as {"idiomatic": {"use-array-flatten": {"flag-wrapped-concat": true}}}
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({
		"description": "Prefer `array.flatten` over `array.concat` with array literals wrapping arguments",
		"location": {
			"col": 6,
			"end": {
				"col": 18,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r if array.concat([a], b)",
		},
	})}
}

test_fail_array_concat_call_with_only_second_argument_wrapped_by_array_literal if {
	r := rule.report with input as ast.policy(`r if array.concat(a, [b])`)
		with config.rules as {"idiomatic": {"use-array-flatten": {"flag-wrapped-concat": true}}}
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {_violation({
		"description": "Prefer `array.flatten` over `array.concat` with array literals wrapping arguments",
		"location": {
			"col": 6,
			"end": {
				"col": 18,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "r if array.concat(a, [b])",
		},
	})}
}

test_success_array_concat_call_with_no_array_literal_arguments if {
	r := rule.report with input as ast.policy(`r if array.concat(a, b)`)
		with config.rules as {"idiomatic": {"use-array-flatten": {"flag-wrapped-concat": true}}}
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == set()
}

test_fail_all_array_concat_calls if {
	r := rule.report with input as ast.policy("r1 if array.concat([1], [2])\nr2 if array.concat([3], [4])")
		with config.rules as {"idiomatic": {"use-array-flatten": {"flag-all-concat": true}}}
		with config.capabilities as {"builtins": {"array.concat": {}, "array.flatten": {}}}

	r == {
		_violation({
			"description": "Prefer using `array.flatten` over `array.concat`",
			"location": {
				"col": 7,
				"end": {
					"col": 19,
					"row": 3,
				},
				"row": 3,
				"text": "r1 if array.concat([1], [2])",
			},
		}),
		_violation({
			"description": "Prefer using `array.flatten` over `array.concat`",
			"location": {
				"col": 7,
				"end": {
					"col": 19,
					"row": 4,
				},
				"row": 4,
				"text": "r2 if array.concat([3], [4])",
			},
		}),
	}
}

_violation(overrides) := object.union(
	{
		"category": "idiomatic",
		"description": "Prefer using `array.flatten` over nested `array.concat` calls",
		"level": "error",
		"location": {
			"col": 0,
			"end": {
				"col": 0,
				"row": 0,
			},
			"file": "policy.rego",
			"row": 0,
			"text": "",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-array-flatten",
		}],
		"title": "use-array-flatten",
	},
	overrides,
)
