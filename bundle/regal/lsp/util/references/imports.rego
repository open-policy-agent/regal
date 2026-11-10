package regal.lsp.util.references

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: |
#   References to the import at the cursor within the same file, i.e. every use of its identifier.
_import_refs contains ref if {
	not _package_target

	_import.path.value[0].value in {"input", "data"}
	identifier := ast.imported_identifier(_import)

	some i, rule in _module.rules
	not _declares_variable(i, identifier)

	some var in _named_vars(rule, identifier)

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(var.location),
		"kind": kind_read,
	}
}

_import_refs contains ref if {
	not _package_target

	_include_declaration

	ref := {
		"uri": input.params.textDocument.uri,
		"range": range.parse(_import.path.location),
		"kind": kind_write,
	}
}

_named_vars(rule, name) := [var |
	walk(rule, [_, var])

	[var.type, var.value] == ["var", name]
]

_declares_variable(rule_index, name) if {
	some kind in ["args", "assign", "some", "somein", "every"]
	ast.found.vars[rule_index][kind][_].value == name
}

_import := find.import_at_position

_import := find.import_alias_at_position

_import := imp if {
	[var, rule_index] := find.var_at_position

	not _declares_variable(rule_index, var.value)

	some imp in _module.imports
	imp.path.value[0].value in {"input", "data"}
	ast.imported_identifier(imp) == var.value
}
