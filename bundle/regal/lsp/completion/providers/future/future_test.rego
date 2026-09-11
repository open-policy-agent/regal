package regal.lsp.completion.providers.future_test

import data.regal.lsp.completion.providers.future as provider
import data.regal.lsp.completion.providers.test_utils as util

test_with_future_keyword if {
	policy := "package policy\n\n"
	module := regal.parse_module("p.rego", policy)
	items := provider.items
		with input as util.input_module_with_location(module, $`{policy}import f`, {"row": 3, "col": 9})
		with data.workspace.config.capabilities.future_keywords as ["not"]

	items == {{
		"detail": "import future `not` keyword",
		"kind": 18,
		"label": "future.keywords.not",
		"labelDetails": {
			"description": "reference",
		},
		"textEdit": {
			"newText": "future.keywords.not",
			"range": {
				"end": {
					"character": 8,
					"line": 2,
				},
				"start": {
					"character": 7,
					"line": 2,
				},
			},
		},
	}}
}

test_no_future_keywords if {
	policy := "package policy\n\n"
	module := regal.parse_module("p.rego", policy)
	items := provider.items
		with input as util.input_module_with_location(module, $`{policy}import f`, {"row": 3, "col": 9})

	items == set()
}
