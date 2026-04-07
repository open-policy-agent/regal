package regal.rules.idiomatic["ambiguous-scope_test"]

import data.regal.ast

import data.regal.rules.idiomatic["ambiguous-scope"] as rule

test_ambiguous_scope_lonely_annotated_rule if {
	module := ast.with_rego_v1(`
# METADATA
# title: allow
allow if input.x

allow if input.y
	`)
	r := rule.report with input as module

	r == {{
		"category": "idiomatic",
		"description": "Ambiguous metadata scope",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 6,
			"end": {
				"col": 15,
				"row": 7,
			},
			"text": "# METADATA",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/ambiguous-scope",
		}],
		"title": "ambiguous-scope",
	}}
}

test_ambiguous_scope_lonely_annotated_function if {
	module := ast.with_rego_v1(`
# METADATA
# title: func
func(x) if x < 10

func(x) if x < "10"
	`)
	r := rule.report with input as module

	r == {{
		"category": "idiomatic",
		"description": "Ambiguous metadata scope",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 6,
			"text": "# METADATA",
			"end": {
				"col": 14,
				"row": 7,
			},
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/ambiguous-scope",
		}],
		"title": "ambiguous-scope",
	}}
}

test_not_ambiguous_scope_all_scoped_to_rule if {
	module := ast.with_rego_v1(`
# METADATA
# description: input has x
allow if input.x

# METADATA
# description: input has y
allow if input.y
	`)
	r := rule.report with input as module

	r == set()
}

test_not_ambiguous_document_scope if {
	module := ast.with_rego_v1(`
# METADATA
# description: input has x or y
# scope: document
allow if input.x

allow if input.y
	`)
	r := rule.report with input as module

	r == set()
}

test_not_ambiguous_entrypoint_exception if {
	module := ast.with_rego_v1(`
# METADATA
# entrypoint: true
allow if input.x

allow if input.y
	`)
	r := rule.report with input as module

	r == set()
}

test_not_ambiguous_both_document_and_rule if {
	module := ast.with_rego_v1(`
# METADATA
# description: input has x or y
# scope: document

# METADATA
# description: input has x
allow if input.x

allow if input.y
	`)
	r := rule.report with input as module

	r == set()
}

test_not_ambiguous_explicit_rule_scope if {
	module := ast.with_rego_v1(`
# METADATA
# description: input has x
# scope: rule
allow if input.x

allow if input.y
	`)
	r := rule.report with input as module

	r == set()
}
