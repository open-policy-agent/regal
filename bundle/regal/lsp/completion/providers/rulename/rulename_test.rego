package regal.lsp.completion.providers.rulename_test

import data.regal.lsp.completion.providers.rulename as provider

test_rule_name_completion[title] if {
	above := "package p\n\n"
	below := "\n\nconstant := 5\n\nfunction(_) := true\n\nrule if 1 + 1 == 3\n\nrule if true\n"
	cache := {"file:///ws/p.rego": regal.parse_module("p.rego", $"{above}{below}")}

	some case in [
		{"name": "all", "typed": "", "expect": ["constant", "function", "rule"]},
		{"name": "constant", "typed": "c", "expect": ["constant"]},
		{"name": "function", "typed": "f", "expect": ["function"]},
		{"name": "rule", "typed": "r", "expect": ["rule"]},
	]

	title := $"typing '{case.typed}' suggests: {concat(", ", case.expect)}"
	items := provider.items with data.workspace.parsed as cache with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": count(case.typed)},
		},
		"regal": {"file": {"lines": split(concat("", [above, case.typed, below]), "\n")}},
	}

	count(items) == count(case.expect)

	some item in items

	item == object.union(_expected[item.label], {"textEdit": {"range": {"end": {"character": count(case.typed)}}}})
}

test_rule_name_completion_only_start_of_line if {
	above := "package p\n\n"
	below := "\n\nconstant := 5\n\nfunction(_) := true\n\nrule if 1 + 1 == 3\n\nrule if true\n"
	cache := {"file:///ws/p.rego": regal.parse_module("p.rego", $"{above}{below}")}
	typed := "foo r"
	items := provider.items with data.workspace.parsed as cache with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": count(typed)},
		},
		"regal": {"file": {"lines": split(concat("", [above, typed, below]), "\n")}},
	}

	count(items) == 0
}

test_rule_name_completion_no_tests if {
	above := "package p\n\n"
	below := "\n\ntest_foo if true\n\n"
	cache := {"file:///ws/p.rego": regal.parse_module("p.rego", $"{above}{below}")}
	typed := "t"
	items := provider.items with data.workspace.parsed as cache with input as {
		"params": {
			"textDocument": {"uri": "file:///ws/p.rego"},
			"position": {"line": 2, "character": count(typed)},
		},
		"regal": {"file": {"lines": split($"{above}{typed}{below}", "\n")}},
	}

	count(items) == 0
}

_expected := {
	"constant": {
		"label": "constant",
		"kind": 21,
		"detail": "rule (constant)",
		"textEdit": {
			"range": {
				"start": {"line": 2, "character": 0},
				"end": {"line": 2, "character": 1000},
			},
			"newText": "constant ",
		},
	},
	"function": {
		"label": "function",
		"kind": 3,
		"detail": "function",
		"textEdit": {
			"range": {
				"start": {"line": 2, "character": 0},
				"end": {"line": 2, "character": 1000},
			},
			"newText": "function ",
		},
	},
	"rule": {
		"label": "rule",
		"kind": 6,
		"detail": "rule",
		"textEdit": {
			"range": {
				"start": {"line": 2, "character": 0},
				"end": {"line": 2, "character": 1000},
			},
			"newText": "rule ",
		},
	},
}
