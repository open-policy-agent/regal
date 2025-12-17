package regal.lsp.completion.providers.ruleheadkeyword_test

import data.regal.lsp.completion.providers.ruleheadkeyword as provider

test_keyword_completion_after_rule_name_no_prefix[label] if {
	items := provider.items with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": 5},
		},
		"regal": {"file": {"lines": split("package p\n\nrule ", "\n")}},
	}

	count(items) == 3

	some label, completion in provider.completions
	expected := object.union(completion, {"textEdit": {
		"newText": $"{label} ",
		"range": {
			"start": {"line": 2, "character": 5},
			"end": {"line": 2, "character": 5},
		},
	}})

	expected in items
}

test_keyword_completion_after_rule_name_i_prefix_suggests_only_if if {
	items := provider.items with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": 6},
		},
		"regal": {"file": {"lines": split("package p\n\nrule i", "\n")}},
	}

	items == {object.union(provider.completions["if"], {"textEdit": {
		"newText": "if ",
		"range": {
			"start": {"line": 2, "character": 5},
			"end": {"line": 2, "character": 6},
		},
	}})}
}

test_completion_after_contains_only_has_if if {
	items := provider.items with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": 18},
		},
		"regal": {"file": {"lines": split("package p\n\nrule contains 100 ", "\n")}},
	}

	expected := {{
		"kind": 14,
		"label": "if",
		"labelDetails": {"description": "add conditions for rule to evaluate"},
		"textEdit": {
			"newText": "if ",
			"range": {
				"end": {"character": 18, "line": 2},
				"start": {"character": 18, "line": 2},
			},
		},
	}}

	items == expected
}
