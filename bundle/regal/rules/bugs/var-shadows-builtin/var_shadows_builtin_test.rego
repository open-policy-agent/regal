package regal.rules.bugs["var-shadows-builtin_test"]

import data.regal.ast
import data.regal.capabilities
import data.regal.config

import data.regal.rules.bugs["var-shadows-builtin"] as rule

test_fail_var_shadows_builtin if {
	module := ast.with_rego_v1(`allow if http := "yes"`)
	r := rule.report with input as module with config.capabilities as capabilities.provided

	r == {{
		"category": "bugs",
		"description": "Variable name shadows built-in",
		"level": "error",
		"location": {
			"col": 10,
			"row": 5,
			"end": {
				"col": 14,
				"row": 5,
			},
			"file": "policy.rego",
			"text": "allow if http := \"yes\"",
		},
		"related_resources": [{
			"description": "documentation",
			"ref": "https://www.openpolicyagent.org/projects/regal/rules/bugs/var-shadows-builtin",
		}],
		"title": "var-shadows-builtin",
	}}
}

test_success_var_does_not_shadow_builtin if {
	r := rule.report with input as ast.policy(`allow if a := "yes"`) with config.capabilities as capabilities.provided

	r == set()
}

# https://github.com/open-policy-agent/regal/issues/1163
test_success_print_excluded if {
	r := rule.report with input as ast.policy(`x if print([y - 1])`) with config.capabilities as capabilities.provided

	r == set()
}
