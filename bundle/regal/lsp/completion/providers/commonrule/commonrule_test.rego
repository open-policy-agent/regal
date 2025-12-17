package regal.lsp.completion.providers.commonrule_test

import data.regal.lsp.completion.providers.commonrule as provider
import data.regal.lsp.completion.providers.test_utils as util

test_common_name_completion_on_invoked if {
	policy := `package policy

import rego.v1


`
	module := regal.parse_module("p.rego", policy)
	items := provider.items with input as util.input_module_with_location(module, policy, {"row": 5, "col": 2})

	expected_item(items, "allow")
	expected_item(items, "deny")
	expected_item(items, "authorized")
}

test_common_name_completion_on_typed if {
	policy := `package policy

import rego.v1


`
	module := regal.parse_module("p.rego", policy)
	items := provider.items with input as util.input_module_with_location(module, $`{policy}d`, {"row": 5, "col": 2})

	expected_item(items, "deny")
}

expected_item(items, label) if {
	item := {
		"label": label,
		"detail": "common name",
		"documentation": {
			"kind": "markdown",
			"value": $`"{label}" is a common rule name`,
		},
		"kind": 15,
		"textEdit": {
			"range": {
				"start": {
					"line": 4,
					"character": 0,
				},
				"end": {
					"line": 4,
					"character": 1,
				},
			},
			"newText": $"{label} ",
		},
	}

	item in items
}
