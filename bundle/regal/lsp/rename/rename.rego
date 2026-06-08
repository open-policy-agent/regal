# METADATA
# description: implementation of the LSP rename feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.newName: {type: "string"}
package regal.lsp.rename

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
default result.response := null

result.response := {"changes": _changes} if _changes != {}

_changes[url] contains change if {
	[arg, rule_index] := find.arg_at_position
	sref := ast.static_prefix(_module.rules[rule_index].head.ref)

	some url
	ref := data.workspace.parsed[url].package.path

	ast.ref_value_equal(_module.package.path, ref)

	some rule in data.workspace.parsed[url].rules

	_has_arg_named(rule.head.args, arg.value)
	ast.ref_value_equal(sref, ast.static_prefix(rule.head.ref))

	some node in ["head", "body", "else"]

	walk(rule[node], [_, value])

	value.type == "var"
	value.value == arg.value

	change := {
		"range": range.parse(value.location),
		"newText": input.params.newName,
	}
}

_has_arg_named(args, name) if {
	some arg in args
	arg.value == name
}

_module := data.workspace.parsed[input.params.textDocument.uri]
