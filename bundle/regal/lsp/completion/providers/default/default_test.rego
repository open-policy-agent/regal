package regal.lsp.completion.providers.default_test

import data.regal.lsp.completion.providers["default"] as provider
import data.regal.lsp.completion.providers.test_utils as util

test_default_completion_on_typing if {
	policy := `package policy

import rego.v1


`
	module := regal.parse_module("p.rego", policy)
	new_policy := $"{policy}d"
	items := provider.items with input as util.input_module_with_location(module, new_policy, {"row": 5, "col": 2})

	items == {{
		"detail": "default <rule-name> := <value>",
		"kind": 14,
		"label": "default",
		"textEdit": {
			"newText": "default ",
			"range": {
				"start": {"character": 0, "line": 4},
				"end": {"character": 1, "line": 4},
			},
		},
	}}
}

test_default_completion_on_typing_with_rule_suggestions if {
	policy := `package policy

import rego.v1

allow if true

deny if false


`
	module := regal.parse_module("p.rego", policy)
	new_policy := $"{policy}d"
	items := provider.items with input as util.input_module_with_location(module, new_policy, {"row": 9, "col": 2})

	items == {
		{
			"detail": "default <rule-name> := <value>",
			"kind": 14,
			"label": "default",
			"textEdit": {
				"newText": "default ",
				"range": {
					"start": {"character": 0, "line": 8},
					"end": {"character": 1, "line": 8},
				},
			},
		},
		{
			"detail": "add default assignment for allow rule",
			"kind": 14,
			"label": "default allow := <value>",
			"textEdit": {
				"newText": "default allow := ",
				"range": {
					"start": {"character": 0, "line": 8},
					"end": {"character": 1, "line": 8},
				},
			},
		},
		{
			"detail": "add default assignment for deny rule",
			"kind": 14,
			"label": "default deny := <value>",
			"textEdit": {
				"newText": "default deny := ",
				"range": {
					"start": {"character": 0, "line": 8},
					"end": {"character": 1, "line": 8},
				},
			},
		},
	}
}

test_default_completion_on_invoked if {
	policy := `package policy

import rego.v1


`
	module := regal.parse_module("p.rego", policy)
	items := provider.items with input as util.input_module_with_location(module, policy, {"row": 5, "col": 2})

	items == {{
		"detail": "default <rule-name> := <value>",
		"kind": 14,
		"label": "default",
		"textEdit": {
			"newText": "default ",
			"range": {
				"start": {"character": 0, "line": 4},
				"end": {"character": 1, "line": 4},
			},
		},
	}}
}
