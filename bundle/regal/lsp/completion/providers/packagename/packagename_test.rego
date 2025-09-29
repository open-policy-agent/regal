package regal.lsp.completion.providers.packagename_test

import data.regal.lsp.completion.providers.packagename as provider

test_package_name_completion_on_typing if {
	policy := `package f`
	provider_input := {
		"params": {
			"textDocument": {"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego"},
			"position": {"line": 0, "character": 9},
		},
		"regal": {
			"file": {
				"lines": split(policy, "\n"),
				"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego",
			},
			"environment": {
				"path_separator": "/",
				"workspace_root_path": "/Users/joe/policy",
			},
		},
	}
	items := provider.items with input as provider_input

	items == {{
		"detail": "suggested package name based on directory structure",
		"kind": 19,
		"label": "foo.bar.baz",
		"textEdit": {
			"newText": "foo.bar.baz\n\n",
			"range": {
				"end": {"character": 9, "line": 0},
				"start": {"character": 8, "line": 0},
			},
		},
	}}
}

test_package_name_completion_on_typing_multiple_suggestions if {
	policy := `package b`
	provider_input := {
		"params": {
			"textDocument": {"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego"},
			"position": {"line": 0, "character": 9},
		},
		"regal": {
			"file": {
				"lines": split(policy, "\n"),
				"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego",
			},
			"environment": {
				"path_separator": "/",
				"workspace_root_path": "/Users/joe/policy",
			},
		},
	}

	items := provider.items with input as provider_input

	items == {
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "bar.baz",
			"textEdit": {
				"newText": "bar.baz\n\n",
				"range": {
					"end": {"character": 9, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "baz",
			"textEdit": {
				"newText": "baz\n\n",
				"range": {
					"end": {"character": 9, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
	}
}

test_package_name_completion_on_typing_multiple_suggestions_when_invoked if {
	policy := `package `
	provider_input := {
		"params": {
			"textDocument": {"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego"},
			"position": {"line": 0, "character": 8},
		},
		"regal": {
			"file": {
				"lines": split(policy, "\n"),
				"uri": "file:///Users/joe/policy/foo/bar/baz/p.rego",
			},
			"environment": {
				"path_separator": "/",
				"workspace_root_path": "/Users/joe/policy",
			},
		},
	}
	items := provider.items with input as provider_input

	items == {
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "foo.bar.baz",
			"textEdit": {
				"newText": "foo.bar.baz\n\n",
				"range": {
					"end": {"character": 8, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "bar.baz",
			"textEdit": {
				"newText": "bar.baz\n\n",
				"range": {
					"end": {"character": 8, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "baz",
			"textEdit": {
				"newText": "baz\n\n",
				"range": {
					"end": {"character": 8, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
	}
}

test_package_name_quoted if {
	policy := `package f`
	provider_input := {
		"params": {
			"textDocument": {"uri": "file:////Users/joe/foo/bar/baz-are/foo/baz-are/foo/p.rego"},
			"position": {"line": 0, "character": 9},
		},
		"regal": {
			"file": {
				"lines": split(policy, "\n"),
				"uri": "file:////Users/joe/foo/bar/baz-are/foo/baz-are/foo/p.rego",
			},
			"environment": {
				"path_separator": "/",
				"workspace_root_path": "/Users/joe/policy",
			},
		},
	}
	items := provider.items with input as provider_input

	items == {
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "foo.bar[\"baz-are\"].foo[\"baz-are\"].foo",
			"textEdit": {
				"newText": "foo.bar[\"baz-are\"].foo[\"baz-are\"].foo\n\n",
				"range": {
					"end": {"character": 9, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "foo",
			"textEdit": {
				"newText": "foo\n\n",
				"range": {
					"end": {"character": 9, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
		{
			"detail": "suggested package name based on directory structure",
			"kind": 19,
			"label": "foo[\"baz-are\"].foo",
			"textEdit": {
				"newText": "foo[\"baz-are\"].foo\n\n",
				"range": {
					"end": {"character": 9, "line": 0},
					"start": {"character": 8, "line": 0},
				},
			},
		},
	}
}

test_suggestions if {
	provider._suggestions("foo.bar.baz", "foo") == ["foo.bar.baz"]
	provider._suggestions("foo.bar.baz", "bar") == ["bar.baz"]
	provider._suggestions("foo.bar.baz", "ba") == ["bar.baz", "baz"]
	provider._suggestions("foo.bar.baz", "") == ["foo.bar.baz", "bar.baz", "baz"]

	# Special character package names filtered at start
	provider._suggestions("foo-bar.baz", "") == ["baz"]
	provider._suggestions("foo@bar.baz", "") == ["baz"]

	# Special characters are quoted
	provider._suggestions("foo.bar-baz.qux", "") == [
		`foo["bar-baz"].qux`,
		"qux",
	]

	provider._suggestions("foo.bar baz.qux", "") == [
		`foo["bar baz"].qux`,
		"qux",
	]

	provider._suggestions("foo.bar@baz.qux", "") == [
		`foo["bar@baz"].qux`,
		"qux",
	]
}

test_needs_quoting if {
	special_chars := ["foo-bar", "foo bar", "foo@bar", "foo.bar", "foo+bar", "föö"]
	every char in special_chars {
		provider._needs_quoting(char) == true
	}

	valid_identifiers := ["foo", "foo_bar", "foo123", "FOO", "foo_123_BAR"]
	every identifier in valid_identifiers {
		provider._needs_quoting(identifier) == false
	}
}
