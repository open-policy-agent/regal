# METADATA
# description: |
#   finds the references to the symbol at the cursor, for the features that need them,
#   like references, rename and document highlight
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.context: {type: "object", properties: {includeDeclaration: {type: "boolean"}}}
package regal.lsp.util.references

import future.keywords.or

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: |
#   every reference to the symbol at input.params.position, with its LSP document highlight kind:
#   Write (3) where the symbol is declared, otherwise Read (2)
# scope: document
refs contains ref if some ref in variable_refs

refs contains ref if some ref in _import_refs

refs contains ref if some ref in rule_refs

refs contains ref if some ref in package_refs

# METADATA
# description: the references to the variable or argument at the cursor
# scope: document
variable_refs := (((_arg_refs | _some_var_refs) | _every_var_refs) | _comprehension_var_refs) | _assign_var_refs

_arg_refs contains ref if {
	[arg, rule_index] := find.arg_at_position
	sref := ast.static_prefix(_module.rules[rule_index].head.ref)

	some url, parsed in _scope
	ast.ref_value_equal(_module.package.path, parsed.package.path)

	some rule in parsed.rules
	arg.value in ast.function_arg_names(rule)
	ast.ref_value_equal(sref, ast.static_prefix(rule.head.ref))

	some node in ["head", "body", "else"]
	walk(rule[node], [_, value])

	value.type == "var"
	value.value == arg.value
	_include_declaration or not value in rule.head.args

	ref := {
		"uri": url,
		"range": range.parse(value.location),
		"kind": _kind_write_if(value in rule.head.args),
	}
}

_some_var_refs contains ref if {
	[var, rule_index] := find.some_var_at_position

	some node in ["head", "body"]
	walk(_module.rules[rule_index][node], [_, value])

	value.type == "var"
	value.value == var.value
	_include_declaration or value != var

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
		"kind": _kind_write_if(value == var),
	}
}

_assign_var_refs contains ref if {
	[var, rule_index] := find.assign_var_at_position

	some node in ["head", "body", "else"]
	walk(_module.rules[rule_index][node], [_, value])

	value.type == "var"
	value.value == var.value
	_include_declaration or not value in ast.found.vars[rule_index].assign

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
		"kind": _kind_write_if(value in ast.found.vars[rule_index].assign),
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
	_include_declaration or value != var

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
		"kind": _kind_write_if(value == var),
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
	_include_declaration or value != var

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(value.location),
		"kind": _kind_write_if(value == var),
	}
}

# METADATA
# description: LSP DocumentHighlightKind of a reference that reads the symbol
kind_read := 2

# METADATA
# description: LSP DocumentHighlightKind of a reference that declares, i.e. writes, the symbol
kind_write := 3

_kind_write_if(true) := kind_write

_kind_write_if(false) := kind_read

default _include_declaration := false

_include_declaration if input.params.context.includeDeclaration

_include_declaration if _renaming

_include_declaration if input.method in {"textDocument/documentHighlight", "textDocument/linkedEditingRange"}

_renaming if input.method in {"textDocument/rename", "textDocument/prepareRename"}

_module := data.workspace.parsed[input.params.textDocument.uri]

# document highlight only needs the references in the current file, so there is no need to
# scan the rest of the workspace for them
_scope := {input.params.textDocument.uri: _module} if {
	input.method == "textDocument/documentHighlight"

	_module
} else := data.workspace.parsed
