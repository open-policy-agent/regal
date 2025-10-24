package regal.lsp.selectionrange_test

import data.regal.lsp.selectionrange

test_selection_range_no_position if {
	result := selectionrange.result.response with input as {"params": {
		"textDocument": {"uri": "file://test.regal"},
		"positions": [],
	}}

	result == null
}

test_selection_range_single_position_rule if {
	result := selectionrange.result.response with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"positions": [{
			"line": 2,
			"character": 7,
		}],
	}}
		with data.workspace.parsed as {"file:///p.rego": regal.parse_module("p.rego", `package p
rule if {
    x := input.foo.bar
}
`)}

	expected := [{
		"parent": {
			"parent": {
				"parent": {"range": {"end": {"character": 1, "line": 3}, "start": {"character": 0, "line": 1}}},
				"range": {"end": {"character": 22, "line": 2}, "start": {"character": 4, "line": 2}},
			},
			"range": {"end": {"character": 8, "line": 2}, "start": {"character": 6, "line": 2}},
		},
		"range": {"end": {"character": 8, "line": 2}, "start": {"character": 6, "line": 2}},
	}]

	result == expected
}

test_selection_range_multiple_positions_rule if {
	result := selectionrange.result.response with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"positions": [
			{"line": 2, "character": 7},
			{"line": 6, "character": 12},
		],
	}}
		with data.workspace.parsed as {"file:///p.rego": regal.parse_module("p.rego", `package p
rule if {
    x := input.foo.bar
}

another if {
    foo := input.baz.qux
}`)}

	expected := [
		{
			"parent": {
				"parent": {
					"parent": {"range": {"end": {"character": 1, "line": 3}, "start": {"character": 0, "line": 1}}},
					"range": {"end": {"character": 22, "line": 2}, "start": {"character": 4, "line": 2}},
				},
				"range": {"end": {"character": 8, "line": 2}, "start": {"character": 6, "line": 2}},
			},
			"range": {"end": {"character": 8, "line": 2}, "start": {"character": 6, "line": 2}},
		},
		{
			"parent": {
				"parent": {
					"parent": {"range": {"end": {"character": 1, "line": 7}, "start": {"character": 0, "line": 5}}},
					"range": {"end": {"character": 24, "line": 6}, "start": {"character": 4, "line": 6}},
				},
				"range": {"end": {"character": 24, "line": 6}, "start": {"character": 11, "line": 6}},
			},
			"range": {"end": {"character": 16, "line": 6}, "start": {"character": 11, "line": 6}},
		},
	]

	result == expected
}

test_selection_range_position_import if {
	result := selectionrange.result.response with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"positions": [{
			"line": 1,
			"character": 15,
		}],
	}}
		with data.workspace.parsed as {"file:///p.rego": regal.parse_module("p.rego", `package p
import data.foo.bar as baz
`)}

	result == [{
		"parent": {
			"parent": {"range": {
				"end": {"character": 26, "line": 1},
				"start": {"character": 0, "line": 1},
			}},
			"range": {"end": {"character": 19, "line": 1}, "start": {"character": 7, "line": 1}},
		},
		"range": {"end": {"character": 15, "line": 1}, "start": {"character": 12, "line": 1}},
	}]
}

test_selection_range_position_package if {
	result := selectionrange.result.response with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"positions": [{
			"line": 0,
			"character": 10,
		}],
	}}
		with data.workspace.parsed as {"file:///p.rego": regal.parse_module("p.rego", `package data.foo.bar.baz
`)}

	result == [{
		"parent": {
			"parent": {"range": {"end": {"character": 24, "line": 0}, "start": {"character": 0, "line": 0}}},
			"range": {"end": {"character": 24, "line": 0}, "start": {"character": 8, "line": 0}},
		},
		"range": {"end": {"character": 12, "line": 0}, "start": {"character": 8, "line": 0}},
	}]
}
