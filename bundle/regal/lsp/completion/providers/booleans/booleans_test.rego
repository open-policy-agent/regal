package regal.lsp.completion.providers.booleans_test

import data.regal.lsp.completion.providers.booleans as provider
import data.regal.lsp.completion.providers.test_utils as utils

test_suggested_in_head if {
	workspace := {"file:///p.rego": `package policy

allow := f`}

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 2, "character": 9},
		},
		"regal": {"file": {"lines": split(workspace["file:///p.rego"], "\n")}},
	}

	items := provider.items with input as regal_module with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 1

	some item in items

	item.label == "false"
}

test_suggested_in_body if {
	workspace := {"file:///p.rego": `package policy

allow if {
  foo := t
}`}

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 9},
		},
		"regal": {"file": {"lines": split(workspace["file:///p.rego"], "\n")}},
	}

	items := provider.items with input as regal_module with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 1

	some item in items

	item.label == "true"
}

test_suggested_after_equals if {
	workspace := {"file:///p.rego": `package policy

allow if {
  foo == t
}`}

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 9},
		},
		"regal": {"file": {"lines": split(workspace["file:///p.rego"], "\n")}},
	}

	items := provider.items with input as regal_module with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 1

	some item in items

	item.label == "true"
}

test_not_suggested_at_start if {
	workspace := {"file:///p.rego": `package policy

allow if {
  t
}`}

	regal_module := {
		"params": {
			"textDocument": {"uri": "file:///p.rego"},
			"position": {"line": 3, "character": 2},
		},
		"regal": {"file": {"lines": split(workspace["file:///p.rego"], "\n")}},
	}

	items := provider.items with input as regal_module with data.workspace.parsed as utils.parsed_modules(workspace)

	count(items) == 0
}
