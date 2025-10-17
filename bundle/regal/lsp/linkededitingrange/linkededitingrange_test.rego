package regal.lsp.linkededitingrange_test

import data.regal.lsp.linkededitingrange

test_linked_editing_range_function_arg if {
	file_content := `package p

foo(bar, baz) := baz if {
    bar == baz
}`

	# querying 'ranges' directly to test without the experimental flag logic
	ranges := linkededitingrange.ranges with input as text_document_position(2, 6)
		with input.regal.file.lines as split(file_content, "\n")
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	expected_ranges := {
		# function arg 'bar' in function head
		{
			"start": {"line": 2, "character": 4},
			"end": {"line": 2, "character": 7},
		},
		# function arg 'bar' reference in function body
		{
			"start": {"line": 3, "character": 4},
			"end": {"line": 3, "character": 7},
		},
	}

	ranges == expected_ranges
}

test_linked_editing_range_disabled_without_flag if {
	file_content := `package p

foo(bar, baz) := baz if {
    bar == baz
}`

	# querying 'ranges' directly to test without the experimental flag logic
	ranges := linkededitingrange.result.response.ranges with input as text_document_position(2, 6)
		with input.regal.file.lines as split(file_content, "\n")
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)

	ranges == set()
}

test_linked_editing_range_enabled_with_flag_set_to_true if {
	file_content := `package p

foo(bar, baz) := baz if {
    bar == baz
}`

	# querying 'ranges' directly to test without the experimental flag logic
	ranges := linkededitingrange.result.response.ranges with input as text_document_position(2, 6)
		with input.regal.file.lines as split(file_content, "\n")
		with data.workspace.parsed["file://p.rego"] as regal.parse_module("p.rego", file_content)
		with opa.runtime as {"env": {"REGAL_EXPERIMENTAL": "true"}}

	expected_ranges := {
		# function arg 'bar' in function head
		{
			"start": {"line": 2, "character": 4},
			"end": {"line": 2, "character": 7},
		},
		# function arg 'bar' reference in function body
		{
			"start": {"line": 3, "character": 4},
			"end": {"line": 3, "character": 7},
		},
	}

	ranges == expected_ranges
}

text_document_position(line, character) := {"params": {
	"textDocument": {"uri": "file://p.rego"},
	"position": {
		"line": line,
		"character": character,
	},
}}
