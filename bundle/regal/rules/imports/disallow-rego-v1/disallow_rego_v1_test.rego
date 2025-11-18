package regal.rules.imports["disallow-rego-v1_test"]

import data.regal.capabilities
import data.regal.config

import data.regal.rules.imports["disallow-rego-v1"] as rule

test_fail_contains_rego_v1_import if {
	r := rule.report with input as regal.parse_module("policy.rego", `package policy
	import rego.v1

	foo if not bar
	`)
		with capabilities.is_opa_v1 as true

	r == {{
		"category": "imports",
		"description": "Use `import rego.v1`",
		"related_resources": [{
			"description": "documentation",
			"ref": config.docs.resolve_url("$baseUrl/$category/disallow-rego-v1", "imports"),
		}],
		"title": "disallow-rego-v1",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 1,
			"end": {
				"col": 8,
				"row": 1,
			},
			"text": "package policy",
		},
		"level": "error",
	}}
}

test_success_no_rego_v1_import if {
	r := rule.report with input as regal.parse_module("policy.rego", `package policy

	foo if not bar
	`)
		with capabilities.is_opa_v1 as true
	r == set()
}

test_success_pre_rego_v1_import if {
	r := rule.report with input as regal.parse_module("policy.rego", `package policy
    import rego.v1

	foo if not bar
	`)
		with capabilities.is_opa_v1 as false
	r == set()
}
