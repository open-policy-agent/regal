package regal.lsp.completion_test

import data.regal.lsp.completion

test_completion_entrypoint if {
	items := completion.items with completion.providers as {"test": {"items": {{"foo": "bar"}}}}

	items == {{"_regal": {"provider": "test"}, "foo": "bar"}}
}

test_inside_comment if {
	_data := {"file:///example.rego": {"comments": [
		{"location": "2:1:2:10"},
		{"location": "4:1:4:10"},
	]}}
	_input := {"regal": {
		"context": {"location": {"row": 4, "col": 5}},
		"file": {"uri": "file:///example.rego"},
	}}

	completion.inside_comment with input as _input with data.workspace.parsed as _data
}

test_not_inside_comment if {
	_data := {"file:///example.rego": {"comments": [
		{"location": "2:1:2:10"},
		{"location": "4:8:4:10"},
	]}}
	_input := {"regal": {
		"context": {"location": {"row": 4, "col": 5}},
		"file": {"uri": "file:///example.rego"},
	}}

	not completion.inside_comment with input as _input with data.workspace.parsed as _data
}
