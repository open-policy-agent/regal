package regal.rules.imports["implicit-future-keywords_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.imports["implicit-future-keywords"] as rule

test_fail_future_keywords_import_wildcard if {
	r := rule.report with input as ast.policy(`import future.keywords`)

	r == {{
		"category": "imports",
		"description": "Use explicit future keyword imports",
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/imports/implicit-future-keywords",
		}],
		"title": "implicit-future-keywords",
		"location": {
			"col": 8,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 14,
				"row": 3,
			},
			"text": `import future.keywords`,
		},
		"level": "error",
	}}
}

test_success_future_keywords_import_specific if {
	r := rule.report with input as ast.policy(`import future.keywords.contains`)

	r == set()
}

test_success_future_keywords_import_specific_many if {
	r := rule.report with input as ast.policy(`
    import future.keywords.contains
    import future.keywords.if
    import future.keywords.in
    `)

	r == set()
}

test_has_notice_if_unmet_capability if {
	r := rule.notices with config.capabilities as {"features": ["rego_v1_import"]}
	r == {{
		"category": "imports",
		"description": "Rule made obsolete by rego.v1 capability",
		"level": "notice",
		"severity": "none",
		"title": "implicit-future-keywords",
	}}
}
