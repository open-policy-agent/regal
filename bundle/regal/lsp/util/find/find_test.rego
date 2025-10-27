package regal.lsp.util.find_test

import data.regal.ast
import data.regal.lsp.util.find

test_rule_at_position[tdp.params.position] if {
	policy := `package p

rule1 if {
	true
}

rule2 if {
	false
}`
	module := regal.parse_module("p.rego", policy)

	some [tdp, exp_name, exp_index] in [
		[text_document_position(2, 3), "rule1", 0],
		[text_document_position(2, 7), "rule1", 0],
		[text_document_position(6, 0), "rule2", 1],
		[text_document_position(6, 10), "rule2", 1],
		[text_document_position(7, 5), "rule2", 1],
	]

	[rule, i] := find.rule_at_position with input as tdp with data.workspace.parsed["file:///p.rego"] as module

	i == exp_index
	rule.head.ref[0].value == exp_name
}

test_import_at_position[tdp.params.position] if {
	policy := `package p
import data.foo.bar
import data.baz.qux
import data.alpha["be-ta"].gamma`

	module := regal.parse_module("p.rego", policy)

	some [tdp, exp_path] in [
		[text_document_position(1, 10), "data.foo.bar"],
		[text_document_position(2, 10), "data.baz.qux"],
		[text_document_position(3, 15), "data.alpha[\"be-ta\"].gamma"],
	]

	imp := find.import_at_position with input as tdp with data.workspace.parsed["file:///p.rego"] as module

	ast.ref_static_to_string(imp.path.value) == exp_path
}

text_document_position(line, character) := {"params": {
	"textDocument": {"uri": "file:///p.rego"},
	"position": {
		"line": line,
		"character": character,
	},
}}
