package regal.rules.style["function-arg-return_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.rules.style["function-arg-return"] as rule

test_fail_function_arg_return_value if {
	r := rule.report with input as ast.policy(`foo := i if { indexof("foo", "o", i) }`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "style",
		"description": "Function argument used for return value",
		"level": "error",
		"location": {
			"col": 35,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 36,
				"row": 3,
			},
			"text": "foo := i if { indexof(\"foo\", \"o\", i) }",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/function-arg-return",
		}],
		"title": "function-arg-return",
	}}
}

test_fail_function_arg_return_value_multi_part_ref if {
	r := rule.report with input as ast.policy(`foo := r if { regex.match("foo", "foo", r) }`)
		with config.capabilities as capabilities.provided

	r == {{
		"category": "style",
		"description": "Function argument used for return value",
		"level": "error",
		"location": {
			"col": 41,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 42,
				"row": 3,
			},
			"text": `foo := r if { regex.match("foo", "foo", r) }`,
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/function-arg-return",
		}],
		"title": "function-arg-return",
	}}
}

test_success_function_arg_return_value_except_function if {
	r := rule.report with input as ast.policy(`foo := i if { indexof("foo", "o", i) }`)
		with config.capabilities as capabilities.provided
		with config.rules as {"style": {"function-arg-return": {"except-functions": ["indexof"]}}}

	r == set()
}
