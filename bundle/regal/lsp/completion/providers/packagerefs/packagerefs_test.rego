package regal.lsp.completion.providers.packagerefs_test

import data.regal.lsp.completion.providers.packagerefs

test_all_package_refs_sugggested_for_import if {
	items := packagerefs.items with data.workspace.parsed as _workspace_parsed with input as {"regal": {
		"context": {"location": {"row": 3, "col": 9}},
		"file": {
			"uri": "file:///example.rego",
			"lines": [
				"package foo.bar",
				"",
				"import d",
			],
		},
	}}

	# 6 suggestions minus the current package and one test package
	# also note how the sortText attribute hints to the client to sort not by
	# the label but by the value calculated based on the number of path components
	# (shortest first) and then alphabetically
	items == {
		_suggestion("data.bar", "000", [2, 7, 2, 8]),
		_suggestion("data.baz", "001", [2, 7, 2, 8]),
		_suggestion("data.foo.baz", "002", [2, 7, 2, 8]),
		_suggestion("data.foo.baz.again", "003", [2, 7, 2, 8]),
	}
}

test_matching_package_refs_sugggested_for_import if {
	items := packagerefs.items with data.workspace.parsed as _workspace_parsed with input as {"regal": {
		"context": {"location": {"row": 3, "col": 14}},
		"file": {
			"uri": "file:///example.rego",
			"lines": [
				"package foo.bar",
				"",
				"import data.f",
			],
		},
	}}

	items == {
		_suggestion("data.foo.baz", "002", [2, 7, 2, 13]),
		_suggestion("data.foo.baz.again", "003", [2, 7, 2, 13]),
	}
}

_to_path(str) := [{"type": _type(i), "value": value} | some i, value in split(str, ".")]

_type(0) := "var"
_type(x) := "string" if x > 0

_suggestion(label, sort_text, range) := {
	"detail": "package",
	"kind": 9,
	"label": label,
	"sortText": sort_text,
	"textEdit": {
		"newText": label,
		"range": {
			"start": {"line": range[0], "character": range[1]},
			"end": {"line": range[2], "character": range[3]},
		},
	},
}

_workspace_parsed := {
	"file:///example.rego": {"package": {"path": _to_path("data.foo.bar")}},
	"file:///other.rego": {"package": {"path": _to_path("data.foo.baz")}},
	"file:///again.rego": {"package": {"path": _to_path("data.foo.baz.again")}},
	"file:///test.rego": {"package": {"path": _to_path("data.foo.baz.again_test")}},
	"file:///more.rego": {"package": {"path": _to_path("data.bar")}},
	"file:///last.rego": {"package": {"path": _to_path("data.baz")}},
}
