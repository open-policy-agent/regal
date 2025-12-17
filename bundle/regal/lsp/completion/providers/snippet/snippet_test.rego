package regal.lsp.completion.providers.snippet_test

import data.regal.lsp.completion.providers.snippet as provider
import data.regal.lsp.completion.providers.test_utils as util

test_snippet_completion_on_typing_partial_prefix if {
	policy := _with_header(`allow if {
	e
}`)
	items := provider.items with input as util.input_with_location(policy, {"row": 6, "col": 2})
	items == {
		{
			"detail": "every key-value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every key-value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:key}, ${2:value} in ${3:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 2, "line": 5},
					"start": {"character": 1, "line": 5},
				},
			},
		},
		{
			"detail": "every value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:var} in ${2:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 2, "line": 5},
					"start": {"character": 1, "line": 5},
				},
			},
		},
	}
}

test_snippet_completion_on_typing_full_prefix if {
	policy := _with_header(`allow if {
	every
}`)
	items := provider.items with input as util.input_with_location(policy, {"row": 6, "col": 6})
	items == {
		{
			"detail": "every key-value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every key-value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:key}, ${2:value} in ${3:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 6, "line": 5},
					"start": {"character": 1, "line": 5},
				},
			},
		},
		{
			"detail": "every value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:var} in ${2:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 6, "line": 5},
					"start": {"character": 1, "line": 5},
				},
			},
		},
	}
}

test_snippet_completion_on_typing_no_repeat if {
	policy := _with_header(`allow if {
	some e in [1,2,3] some
}`)

	items := provider.items with input as util.input_with_location(policy, {"row": 6, "col": 21})
	items == set()
}

test_no_snippet_completion_on_[typed] if {
	template := `allow if {
	%s
}`

	some typed in ["i", "in", "inp", "inpu", "input", "input.", "input.x"]

	policy := _with_header(sprintf(template, [typed]))

	items := provider.items with input as util.input_with_location(policy, {"row": 6, "col": 1 + count(typed)})
	items == set()
}

test_snippet_completion_on_invoked if {
	items := provider.items with input as util.input_with_location(_with_header(`allow if `), {"row": 5, "col": 10})
	items == {
		{
			"detail": "every key-value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every key-value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:key}, ${2:value} in ${3:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 9, "line": 4},
					"start": {"character": 9, "line": 4},
				},
			},
		},
		{
			"detail": "every value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "every value iteration (snippet)",
			"textEdit": {
				"newText": "every ${1:var} in ${2:collection} {\n\t$0\n}",
				"range": {
					"end": {"character": 9, "line": 4},
					"start": {"character": 9, "line": 4},
				},
			},
		},
		{
			"detail": "some key-value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "some key-value iteration (snippet)",
			"textEdit": {
				"newText": "some ${1:key}, ${2:value} in ${3:collection}\n$0",
				"range": {
					"end": {"character": 9, "line": 4},
					"start": {"character": 9, "line": 4},
				},
			},
		},
		{
			"detail": "some value iteration",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "some value iteration (snippet)",
			"textEdit": {
				"newText": "some ${1:var} in ${2:collection}\n$0",
				"range": {
					"end": {"character": 9, "line": 4},
					"start": {"character": 9, "line": 4},
				},
			},
		},
	}
}

test_metadata_snippet_completion if {
	items := provider.items with input as util.input_with_location(_with_header(``), {"row": 5, "col": 1})
	items == {
		{
			"detail": "metadata annotation",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "metadata annotation [description] (snippet)",
			"textEdit": {
				"newText": "# METADATA\n# description: ${1:description}",
				"range": {
					"end": {"character": 0, "line": 4},
					"start": {"character": 0, "line": 4},
				},
			},
		},
		{
			"detail": "metadata annotation",
			"insertTextFormat": 2,
			"kind": 15,
			"label": "metadata annotation [title, description] (snippet)",
			"textEdit": {
				"newText": "# METADATA\n# title: ${1:title}\n# description: ${2:description}",
				"range": {
					"end": {"character": 0, "line": 4},
					"start": {"character": 0, "line": 4},
				},
			},
		},
	}
}

_with_header(policy) := $`package policy

import rego.v1

{policy}`
