# METADATA
# description: implementation of the LSP references feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.context: {type: "object", properties: {includeDeclaration: {type: "boolean"}}}
package regal.lsp.references

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
result.response contains ref if some ref in _arg_refs

result.response contains ref if some ref in _some_var_refs

_arg_refs contains ref if {
	[arg, rule_index] := find.arg_at_position
	sref := ast.static_prefix(_module.rules[rule_index].head.ref)

	some url, parsed in data.workspace.parsed
	ast.ref_value_equal(_module.package.path, parsed.package.path)

	some rule in parsed.rules
	_has_arg_named(rule.head.args, arg.value)
	ast.ref_value_equal(sref, ast.static_prefix(rule.head.ref))

	some node in ["head", "body", "else"]
	walk(rule[node], [_, value])

	value.type == "var"
	value.value == arg.value

	ref := {
		"uri": url,
		"range": range.parse(value.location),
	}
}

_some_var_refs contains ref if {
	[var, rule_index] := find.some_var_at_position

	walk(_module.rules[rule_index].body, [_, value])

	value.type == "var"
	value.value == var.value

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
	}
}

_has_arg_named(args, name) if {
	some arg in args
	arg.value == name
}

_module := data.workspace.parsed[input.params.textDocument.uri]
