package regal.lsp.completion_test

import data.regal.lsp.completion

test_completion_entrypoint if {
	items := completion.items with completion.providers as {"test": {"items": {{"foo": "bar"}}}}

	items == {{"foo": "bar"}}
}

test_inside_comment if completion.inside_comment
	with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"position": {"line": 3, "character": 4},
	}}
	with data.workspace.parsed as {"file:///p.rego": {"comments": [
		{"location": "2:1:2:10"},
		{"location": "4:1:4:10"},
	]}}

test_not_inside_comment if not completion.inside_comment
	with input as {"params": {
		"textDocument": {"uri": "file:///p.rego"},
		"position": {"line": 3, "character": 4},
	}}
	with data.workspace.parsed as {"file:///p.rego": {"comments": [
		{"location": "2:1:2:10"},
		{"location": "4:8:4:10"},
	]}}
