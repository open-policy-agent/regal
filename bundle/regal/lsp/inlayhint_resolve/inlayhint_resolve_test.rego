package regal.lsp.inlayhint_resolve_test

import data.regal.lsp.inlayhint_resolve

test_inlayhint_tooltip_resolve_with_description if {
	res := inlayhint_resolve.result.response with input.params as {
		"kind": 2,
		"label": "base:",
		"paddingRight": true,
		"position": {
			"character": 13,
			"line": 3,
		},
		"data": {
			"name": "base",
			"type": "string",
			"description": "base string",
		},
	}

	res == {
		"kind": 2,
		"label": "base:",
		"paddingRight": true,
		"position": {
			"character": 13,
			"line": 3,
		},
		"tooltip": {"kind": "markdown", "value": "`base` — string: base string"},
	}
}

test_inlayhint_tooltip_resolve_no_description if {
	res := inlayhint_resolve.result.response with input.params as {
		"kind": 2,
		"label": "base:",
		"paddingRight": true,
		"position": {
			"character": 13,
			"line": 3,
		},
		"data": {
			"name": "base",
			"type": "string",
		},
	}

	res == {
		"kind": 2,
		"label": "base:",
		"paddingRight": true,
		"position": {
			"character": 13,
			"line": 3,
		},
		"tooltip": {"kind": "markdown", "value": "`base` — string"},
	}
}
