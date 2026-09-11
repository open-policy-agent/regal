package regal.lsp.completion.providers.import_test

import data.regal.lsp.completion.providers["import"] as provider
import data.regal.lsp.completion.providers.test_utils as util

test_import_completion_empty_line if {
	policy := `package policy

import rego.v1

`
	items := provider.items with input as util.input_with_location(policy, {"row": 5, "col": 1})
	items == {{
		"label": "import",
		"detail": "import <path>",
		"kind": 14,
		"documentation": {
			"kind": "markdown",
			"value": provider.doc,
		},
		"textEdit": {
			"newText": "import ",
			"range": {
				"start": {"character": 0, "line": 4},
				"end": {"character": 0, "line": 4},
			},
		},
	}}
}

test_import_completion_on_typing if {
	policy := `package policy

import rego.v1

imp`

	items := provider.items with input as util.input_with_location(policy, {"row": 5, "col": 3})

	items == {{
		"label": "import",
		"detail": "import <path>",
		"kind": 14,
		"documentation": {
			"kind": "markdown",
			"value": provider.doc,
		},
		"textEdit": {
			"newText": "import ",
			"range": {
				"start": {"character": 0, "line": 4},
				"end": {"character": 3, "line": 4},
			},
		},
	}}
}
