package regal.rules.style["opa-fmt_test"]

import data.regal.rules.style["opa-fmt"] as rule

test_fail_not_formatted if {
	r := rule.report with input as regal.parse_module("p.rego", `package p    `)
		with input.regal.file.rego_version as "v1"

	r == {{
		"category": "style",
		"description": "File should be formatted with `opa fmt`",
		"level": "error",
		"location": {
			"col": 1,
			"file": "p.rego",
			"row": 1,
			"text": "package p    ",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/opa-fmt",
		}],
		"title": "opa-fmt",
	}}
}

test_success_formatted if {
	r := rule.report with input as regal.parse_module("p.rego", "package p\n")
		with input.regal.file.rego_version as "v1"

	r == set()
}

test_fail_v0_required_but_v1_policy if {
	r := rule.report with input as regal.parse_module("p.rego", "package p\n\none contains 1\n")
		with input.regal.file.rego_version as "v0"

	r == {{
		"category": "style",
		"description": "File should be formatted with `opa fmt`",
		"level": "error",
		"location": {
			"col": 1,
			"file": "p.rego",
			"row": 1,
			"text": "package p",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/style/opa-fmt",
		}],
		"title": "opa-fmt",
	}}
}

test_success_v0_required_and_v0_policy if {
	r := rule.report with input as regal.parse_module("p_v0.rego", "package p\n\none[\"1\"]\n")
		with input.regal.file.rego_version as "v0"

	r == set()
}
