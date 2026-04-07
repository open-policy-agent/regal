package regal.rules.idiomatic["custom-has-key-construct_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.idiomatic["custom-has-key-construct"] as rule

test_fail_custom_has_key if {
	r := rule.report with input as ast.policy(`has_key(name, coll) if {
		_ = coll[name]
	}`)

	r == {{
		"category": "idiomatic",
		"description": "Custom function may be replaced by `in` and `object.keys`",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 20,
				"row": 3,
			},
			"text": "has_key(name, coll) if {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-has-key-construct",
		}],
		"title": "custom-has-key-construct",
	}}
}

test_fail_custom_has_key_reversed if {
	r := rule.report with input as ast.policy(`has_key(name, coll) if {
		coll[name] = _
	}`)

	r == {{
		"category": "idiomatic",
		"description": "Custom function may be replaced by `in` and `object.keys`",
		"level": "error",
		"location": {
			"col": 1,
			"file": "policy.rego",
			"row": 3,
			"end": {
				"col": 20,
				"row": 3,
			},
			"text": "has_key(name, coll) if {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-has-key-construct",
		}],
		"title": "custom-has-key-construct",
	}}
}

test_fail_custom_has_key_multiple_wildcards if {
	r := rule.report with input as ast.policy(`
	other_rule contains "foo" if {
		wildcard := input[_]
	}

	has_key(name, coll) if {
		coll[name] = _
	}`)

	r == {{
		"category": "idiomatic",
		"description": "Custom function may be replaced by `in` and `object.keys`",
		"level": "error",
		"location": {
			"col": 2,
			"file": "policy.rego",
			"row": 8,
			"end": {
				"col": 21,
				"row": 8,
			},
			"text": "\thas_key(name, coll) if {",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/idiomatic/custom-has-key-construct",
		}],
		"title": "custom-has-key-construct",
	}}
}

test_has_notice_if_unmet_capability if {
	r := rule.notices with config.capabilities as {}
	r == {{
		"category": "idiomatic",
		"description": "Missing capability for built-in function `object.keys`",
		"level": "notice",
		"severity": "warning",
		"title": "custom-has-key-construct",
	}}
}
