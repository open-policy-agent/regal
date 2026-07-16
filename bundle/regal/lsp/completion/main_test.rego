package regal.lsp.completion_test

import data.regal.ast
import data.regal.util

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

# Ensure that completions are provided both when itemDefaults are supported
# by the client and when they are not. We use the boolean completion provider
# below for it's simplicity only — it's the main logic that's being tested
# here and nothing else
test_edit_range_support_result[supported] if {
	module := ast.policy("r := f")

	some item_defaults in [[], ["editRange"]]

	supported := count(item_defaults) == 1

	result := completion.items
		with data.client.capabilities.textDocument.completion.completionList.itemDefaults as item_defaults
		with data.workspace["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position.line as 2
		with input.params.position.character as count("r := f")
		with input.regal as module.regal

	util.single_set_item(result).label == "false"
}
