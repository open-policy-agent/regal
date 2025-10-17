package regal.lsp.documenthighlight_test

import data.regal.lsp.documenthighlight

test_metadata_header_highlight if {
	file_content := `package p

# METADATA
# description: A test rule
# scope: document
# title: Test Rule
allow if true`

	items := documenthighlight.items with input as {
		"params": {
			"textDocument": {"uri": "file://p.rego"},
			"position": {"line": 2, "character": 5},
		},
		"regal": {"file": {"lines": split(file_content, "\n")}},
	}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	items == {
		# the METADATA itself
		{
			"range": {
				"start": {"line": 2, "character": 2},
				"end": {"line": 2, "character": 10},
			},
			"kind": 1,
		},
		# description
		{
			"range": {
				"start": {"line": 3, "character": 2},
				"end": {"line": 3, "character": 13},
			},
			"kind": 1,
		},
		# scope
		{
			"range": {
				"start": {"line": 4, "character": 2},
				"end": {"line": 4, "character": 7},
			},
			"kind": 1,
		},
		# title
		{
			"range": {
				"start": {"line": 5, "character": 2},
				"end": {"line": 5, "character": 7},
			},
			"kind": 1,
		},
	}
}

test_individual_metadata_attribute_highlight if {
	file_content := `package p

# METADATA
# description: A test rule
# scope: document
# title: Test Rule
allow if true`

	items := documenthighlight.items with input as {
		"params": {
			"textDocument": {"uri": "file://p.rego"},
			# over the description attribute
			"position": {"line": 3, "character": 5},
		},
		"regal": {"file": {"lines": split(file_content, "\n")}},
	}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	# should highlight only the description attribute clicked
	items == {{
		"range": {
			"start": {"line": 3, "character": 2},
			"end": {"line": 3, "character": 13},
		},
		"kind": 1,
	}}
}

test_arg_highlight_in_head_and_body[name] if {
	file_content := `package p

fun(user, action, resource) := user if {
	user == "alice"
	action.detail == "detail"
	resource.type == input.nested[resource.foo]
}`

	some [name, inp, exp] in [
		["user", {"line": 2, "character": 6}, {
			{
				"kind": 2,
				"range": {"end": {"character": 8, "line": 2}, "start": {"character": 4, "line": 2}},
			},
			{
				"kind": 3,
				"range": {"end": {"character": 5, "line": 3}, "start": {"character": 1, "line": 3}},
			},
			{
				"kind": 3,
				"range": {"end": {"character": 35, "line": 2}, "start": {"character": 31, "line": 2}},
			},
		}],
		["action", {"line": 2, "character": 12}, {
			{
				"kind": 2,
				"range": {"end": {"character": 16, "line": 2}, "start": {"character": 10, "line": 2}},
			},
			{
				"kind": 3,
				"range": {"end": {"character": 7, "line": 4}, "start": {"character": 1, "line": 4}},
			},
		}],
		["resource", {"line": 2, "character": 19}, {
			{
				"kind": 2,
				"range": {"end": {"character": 26, "line": 2}, "start": {"character": 18, "line": 2}},
			},
			{
				"kind": 3,
				"range": {"end": {"character": 9, "line": 5}, "start": {"character": 1, "line": 5}},
			},
			{
				"kind": 3,
				"range": {"end": {"character": 39, "line": 5}, "start": {"character": 31, "line": 5}},
			},
		}],
	]

	items := documenthighlight.items with input as {
		"params": {"textDocument": {"uri": "file://p.rego"}, "position": inp},
		"regal": {"file": {"lines": split(file_content, "\n")}},
	}
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	items == exp
}
