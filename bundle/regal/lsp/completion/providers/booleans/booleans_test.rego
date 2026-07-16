package regal.lsp.completion.providers.booleans_test

import data.regal.ast
import data.regal.util

import data.regal.lsp.completion.providers.booleans as provider

test_boolean_suggested_where_expected[$"{name}-{count(item_defaults)}"] if {
	# assert that boolean suggestions are provided at all appropriate locations,
	# and that the format of the suggestion is correct both with and without the
	# `editRange` item default in the client's capability list. The latter property
	# must be reflected in each test's name or else both cases will be counted as
	# one (as the test name is the key in the result set).
	some item_defaults in [[], ["editRange"]]
	some [name, edit] in [
		["head key literal", "rule[t█] if foo"],
		["head value assign", "rule := t█"],
		["body expression start", "rule if t█"],
		["body expression compare", "rule if x == t█"],
		["function argument first", "f(t█)"],
		["function argument next", "f(foo, t█)"],
		["object literal", "o := {\"key\": t█}"],
		["array item first", "a := [t█]"],
		["array item next", "a := [foo, t█]"],
	]

	module := ast.policy(replace(edit, "█", ""))
	result := provider.items
		with data.client.capabilities.textDocument.completion.completionList.itemDefaults as item_defaults
		with data.workspace["file:///p.rego"] as module
		with input.params.textDocument.uri as "file:///p.rego"
		with input.params.position.line as 2
		with input.params.position.character as indexof(edit, "█")
		with input.regal as module.regal

	item := util.single_set_item(result)

	item.label == "true"
	item.labelDetails.description == "value"
	item.detail == "boolean value"
	item.kind == 12

	has_text_edit := object.get(item, "textEdit", count(item_defaults) == 1)
	has_text_edit != false
}
