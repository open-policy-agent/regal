package regal.rules.idiomatic["use-object-keys_test"]

import data.regal.ast

import data.regal.rules.idiomatic["use-object-keys"] as rule

test_fail_use_object_keys_not_some_in_comprehension if {
	r := rule.report with input as ast.policy(`comp := {k | some k, _ in input.object}`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer to use `object.keys`",
		"level": "error",
		"location": {
			"col": 9,
			"end": {
				"col": 40,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "comp := {k | some k, _ in input.object}",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-object-keys",
		}],
		"title": "use-object-keys",
	}}
}

test_fail_use_object_keys_not_some_in_comprehension_single_var if {
	r := rule.report with input as ast.policy(`comp := {k | some k, _ in object}`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer to use `object.keys`",
		"level": "error",
		"location": {
			"col": 9,
			"end": {
				"col": 34,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "comp := {k | some k, _ in object}",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-object-keys",
		}],
		"title": "use-object-keys",
	}}
}

test_fail_use_object_keys_not_some_comprehension if {
	r := rule.report with input as ast.policy(`comp := {k | some k; input.object[k]}`)

	r == {{
		"category": "idiomatic",
		"description": "Prefer to use `object.keys`",
		"level": "error",
		"location": {
			"col": 9,
			"end": {
				"col": 38,
				"row": 3,
			},
			"file": "policy.rego",
			"row": 3,
			"text": "comp := {k | some k; input.object[k]}",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/use-object-keys",
		}],
		"title": "use-object-keys",
	}}
}
