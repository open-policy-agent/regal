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

result.response contains ref if some ref in _every_var_refs

result.response contains ref if some ref in _comprehension_var_refs

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

	some node in ["head", "body"]
	walk(_module.rules[rule_index][node], [_, value])

	value.type == "var"
	value.value == var.value

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
	}
}

# METADATA
# description: |
#   References to the `every`-declared variable at the cursor: the declaration
#   site (key or value position) plus every reference inside the block's body.
_every_var_refs contains ref if {
	[var, every_terms] := find.every_var_at_position

	some node in ["body", "key", "value"]
	walk(every_terms[node], [_, value])

	value.type == "var"
	value.value == var.value

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
	}
}

# METADATA
# description: |
#   References to the variable declared via `some x in y` inside a
#   comprehension. Walking the entire comprehension covers both the
#   declaration and every reference, scoped to that comprehension only.
_comprehension_var_refs contains ref if {
	[var, comp] := find.comprehension_var_at_position

	walk(comp, [_, value])

	value.type == "var"
	value.value == var.value
	not startswith(value.value, "$")

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
