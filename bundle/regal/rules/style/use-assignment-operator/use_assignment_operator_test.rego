package regal.rules.style["use-assignment-operator_test"]

import data.regal.ast
import data.regal.rules.style["use-assignment-operator"] as rule

test_fail_unification_in_regular_assignment if {
	r := rule.report with input as ast.policy(`foo = false`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 5,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 6,
				"row": 3,
			},
			"text": "foo = false",
		},
		"level": "error",
	}}
}

test_fail_not_implicit_boolean_assignment_with_body if {
	r := rule.report with input as ast.policy(`allow = true if { true }`)

	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 7,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 8,
				"row": 3,
			},
			"text": "allow = true if { true }",
		},
		"level": "error",
	}}
}

test_fail_not_implicit_boolean_assignment if {
	r := rule.report with input as ast.policy(`foo = true`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 5,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 6,
				"row": 3,
			},
			"text": "foo = true",
		},
		"level": "error",
	}}
}

test_success_implicit_boolean_assignment if {
	r := rule.report with input as ast.policy(`allow if { true }`)
	r == set()
}

test_fail_unification_in_default_assignment if {
	r := rule.report with input as ast.policy(`default x = false`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 11,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 12,
				"row": 3,
			},
			"text": "default x = false",
		},
		"level": "error",
	}}
}

test_fail_unification_in_default_function_assignment if {
	r := rule.report with input as ast.policy(`default x(_) = false`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 14,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 15,
				"row": 3,
			},
			"text": "default x(_) = false",
		},
		"level": "error",
	}}
}

test_success_assignment_in_default_assignment if {
	r := rule.report with input as ast.policy(`default x := false`)
	r == set()
}

test_success_assignment_in_default_function_assignment if {
	r := rule.report with input as ast.policy(`default x(_) := false`)
	r == set()
}

test_fail_unification_in_object_rule_assignment if {
	r := rule.report with input as ast.policy(`x["a"] = 1`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 9,
				"row": 3,
			},
			"text": `x["a"] = 1`,
		},
		"level": "error",
	}}
}

test_success_assignment_in_object_rule_assignment if {
	r := rule.report with input as ast.policy(`x["a"] := 1`)
	r == set()
}

test_fail_unification_in_function_assignment if {
	r := rule.report with input as ast.policy(`foo(bar) = "baz"`)
	r == {{
		"category": "style",
		"description": "Prefer := over = for assignment",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
		}],
		"title": "use-assignment-operator",
		"location": {
			"col": 10,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 11,
				"row": 3,
			},
			"text": `foo(bar) = "baz"`,
		},
		"level": "error",
	}}
}

test_success_implicit_boolean_assignment_function if {
	r := rule.report with input as ast.policy(`f(x) if { 1 == 1 }`)
	r == set()
}

test_success_assignment_operator_function if {
	r := rule.report with input as ast.policy(`f(x) := true if { 1 == 1 }`)
	r == set()
}

test_success_partial_rule if {
	r := rule.report with input as ast.policy(`partial["works"] if { 1 == 1 }`)
	r == set()
}

test_success_using_if if {
	r := rule.report with input as ast.policy(`foo if 1 == 1`)

	r == set()
}

test_success_ref_head_rule_if if {
	r := rule.report with input as ast.policy(`a.b.c if true`)

	r == set()
}

test_success_ref_head_rule_with_var_if if {
	r := rule.report with input as ast.policy(`works[x] if x := 5`)

	r == set()
}

test_fail_unification_in_else if {
	r := rule.report with input as ast.policy(`
	allow if {
		input.x
	} else = true if {
		input.y
	} else = false
	`)
	r == {
		{
			"category": "style",
			"description": "Prefer := over = for assignment",
			"related_resources": [{
				"description": "documentation",
				"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
			}],
			"title": "use-assignment-operator",
			"location": {"col": 9, "file": "policy.rego", "row": 6, "text": "\t} else = true if {", "end": {
				"col": 10,
				"row": 6,
			}},
			"level": "error",
		},
		{
			"category": "style",
			"description": "Prefer := over = for assignment",
			"related_resources": [{
				"description": "documentation",
				"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/use-assignment-operator",
			}],
			"title": "use-assignment-operator",
			"location": {"col": 9, "file": "policy.rego", "row": 8, "text": "\t} else = false", "end": {
				"col": 10,
				"row": 8,
			}},
			"level": "error",
		},
	}
}

test_success_assignment_in_else if {
	r := rule.report with input as ast.policy(`
	allow if {
		input.x
	} else := true if {
		input.y
	} else if {
		input.z
	} else := false
	`)
	r == set()
}
