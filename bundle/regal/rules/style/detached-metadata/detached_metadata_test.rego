package regal.rules.style["detached-metadata_test"]

import data.regal.ast
import data.regal.rules.style["detached-metadata"] as rule

test_fail_detached_metadata if {
	r := rule.report with input as regal.parse_module("p.rego", `
package p

# METADATA
# description: allow is always true

allow := true
`)

	r == {{
		"category": "style",
		"description": "Detached metadata annotation",
		"level": "error",
		"location": {
			"col": 1,
			"file": "p.rego",
			"row": 4,
			"end": {
				"col": 36,
				"row": 5,
			},
			"text": "# METADATA",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/detached-metadata",
		}],
		"title": "detached-metadata",
	}}
}

test_fail_very_detached_metadata if {
	r := rule.report with input as regal.parse_module("p.rego", `
package p

# METADATA
# description: allow is always true



allow := true
`)
	r == {{
		"category": "style",
		"description": "Detached metadata annotation",
		"level": "error",
		"location": {
			"col": 1,
			"file": "p.rego",
			"row": 4,
			"end": {
				"col": 36,
				"row": 5,
			},
			"text": "# METADATA",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/detached-metadata",
		}],
		"title": "detached-metadata",
	}}
}

test_success_attached_metadata if {
	r := rule.report with input as ast.with_rego_v1(`
# METADATA
# title: valid
allow := true
`)
	r == set()
}

test_success_detached_but_more_metadata_on_rule if {
	r := rule.report with input as ast.with_rego_v1(`
# METADATA
# scope: document
# description: allow allows

# METADATA
# title: allow
allow := true
`)
	r == set()
}

test_success_detached_but_more_metadata_on_package if {
	r := rule.report with input as regal.parse_module("p.rego", `
# METADATA
# scope: package
# description: foo

# METADATA
# title: allow
# scope: subpackages
package p
`)
	r == set()
}

test_fail_second_block_detached_first_not_reported if {
	r := rule.report with input as regal.parse_module("p.rego", `# METADATA
# scope: package
# description: foo

# METADATA
# title: allow
# scope: subpackages

package p
`)
	r == {{
		"category": "style",
		"description": "Detached metadata annotation",
		"level": "error",
		"location": {
			"col": 1,
			"file": "p.rego",
			"row": 5,
			"end": {
				"col": 21,
				"row": 7,
			},
			"text": "# METADATA",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/detached-metadata",
		}],
		"title": "detached-metadata",
	}}
}

test_success_not_detached_by_comment_in_different_column if {
	r := rule.report with input as ast.with_rego_v1(`
# METADATA
# title: allow
allow := true # not in block
`)
	r == set()
}
