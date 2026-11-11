package regal.lsp.linkededitingrange_test

import data.regal.lsp.linkededitingrange

test_linked_editing_range_function_arg if {
	file_content := `package p

foo(bar, baz) := baz if {
    bar == baz
}`

	ranges := linkededitingrange.result.response.ranges
		with input as text_document_position(2, 6)
		with input.method as "textDocument/linkedEditingRange"
		with input.regal.file.lines as split(file_content, "\n")
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	expected_ranges := {
		# function arg 'bar' in function head
		{
			"start": {
				"line": 2,
				"character": 4,
			},
			"end": {
				"line": 2,
				"character": 7,
			},
		},
		# function arg 'bar' reference in function body
		{
			"start": {
				"line": 3,
				"character": 4,
			},
			"end": {
				"line": 3,
				"character": 7,
			},
		},
	}

	ranges == expected_ranges
}

test_linked_editing_range_function_arg_links_all_heads_in_file_only if {
	policy := `package p

foo(bar) := 1 if bar == 1

foo(bar) := 2 if bar == 2`
	other_file_policy := `package p





foo(bar) := 3 if bar == 3`

	ranges := linkededitingrange.ranges
		with input as text_document_position(2, 4)
		with input.method as "textDocument/linkedEditingRange"
		with input.regal.file.lines as split(policy, "\n")
		with data.workspace.parsed as {
			"file://p.rego": regal.parse_module("p.rego", policy),
			"file://other.rego": regal.parse_module("other.rego", other_file_policy),
		}

	ranges == {
		{"start": {"line": 2, "character": 4}, "end": {"line": 2, "character": 7}},
		{"start": {"line": 2, "character": 17}, "end": {"line": 2, "character": 20}},
		{"start": {"line": 4, "character": 4}, "end": {"line": 4, "character": 7}},
		{"start": {"line": 4, "character": 17}, "end": {"line": 4, "character": 20}},
	}
}

test_linked_editing_range_links_declared_variable_and_its_uses if {
	policy := `package p

foo if {
	x := input.a
	x == 1
}`

	ranges := linkededitingrange.ranges
		with input as text_document_position(3, 1)
		with input.method as "textDocument/linkedEditingRange"
		with input.regal.file.lines as split(policy, "\n")
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", policy)

	ranges == {
		{"start": {"line": 3, "character": 1}, "end": {"line": 3, "character": 2}},
		{"start": {"line": 4, "character": 1}, "end": {"line": 4, "character": 2}},
	}
}

test_linked_editing_range_rule_used_only_in_file_links_heads_and_uses if {
	policy := `package p

allow if input.admin

allow if input.owner

other if allow`
	other_file_policy := `package q

x := 1`

	ranges := linkededitingrange.result.response.ranges
		with input as text_document_position(2, 1)
		with input.method as "textDocument/linkedEditingRange"
		with input.regal.file.lines as split(policy, "\n")
		with data.workspace.parsed as {
			"file://p.rego": regal.parse_module("p.rego", policy),
			"file://other.rego": regal.parse_module("other.rego", other_file_policy),
		}

	ranges == {
		{"start": {"line": 2, "character": 0}, "end": {"line": 2, "character": 5}},
		{"start": {"line": 4, "character": 0}, "end": {"line": 4, "character": 5}},
		{"start": {"line": 6, "character": 9}, "end": {"line": 6, "character": 14}},
	}
}

test_linked_editing_range_rule_with_references_in_other_file_is_not_linked[name] if {
	policy := `package p

allow if input.admin`

	some name, other_file_policy in {
		"use": "package q\n\nimport data.p\n\nx if p.allow",
		"head": "package p\n\nallow if input.owner",
	}

	ranges := linkededitingrange.result.response.ranges
		with input as text_document_position(2, 1)
		with input.method as "textDocument/linkedEditingRange"
		with input.regal.file.lines as split(policy, "\n")
		with data.workspace.parsed as {
			"file://p.rego": regal.parse_module("p.rego", policy),
			"file://other.rego": regal.parse_module("other.rego", other_file_policy),
		}

	ranges == set()
}

text_document_position(line, character) := {"params": {
	"textDocument": {"uri": "file://p.rego"},
	"position": {
		"line": line,
		"character": character,
	},
}}
