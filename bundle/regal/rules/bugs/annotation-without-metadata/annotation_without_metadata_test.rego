package regal.rules.bugs["annotation-without-metadata_test"]

import data.regal.ast

import data.regal.rules.bugs["annotation-without-metadata"] as rule

test_fail_annotation_without_metadata if {
	r := rule.report with input as ast.with_rego_v1(`
# title: allow
allow := false
	`)

	r == {{
		"category": "bugs",
		"description": "Annotation without metadata",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 6,
			"end": {
				"col": 15,
				"row": 6,
			},
			"text": "# title: allow",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/annotation-without-metadata",
		}],
		"title": "annotation-without-metadata",
	}}
}

test_success_annotation_with_metadata if {
	r := rule.report with input as ast.policy(`
# METADATA
# title: allow
allow := false
	`)

	r == set()
}

test_success_annotation_but_no_metadata_location if {
	r := rule.report with input as ast.policy(`allow := false # title: allow`)

	r == set()
}

test_success_annotation_without_metadata_but_comment_preceding if {
	r := rule.report with input as ast.policy(`
# something that is not an annotation here will cancel this rule
# as this is less likely to be a mistake... but weird
# title: allow
allow := false
	`)

	r == set()
}
