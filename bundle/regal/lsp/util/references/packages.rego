package regal.lsp.util.references

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: |
#   References to the package at the cursor: where it is imported in any file, and
#   its package declarations when declarations are included.
_package_refs contains ref if {
	some [file, _, imp] in _package_imports

	ref := {
		"uri": file,
		"range": range.parse(imp.path.location),
		"kind": kind_read,
	}
}

# uses of the import name
_package_refs contains ref if {
	some [file, module, imp] in _package_imports

	id := ast.imported_identifier(imp)

	some rule in module.rules
	not _declares_local(rule, id)

	some var in _named_vars(rule, id)

	ref := {
		"uri": file,
		"range": range.parse(var.location),
		"kind": kind_read,
	}
}

# the package declarations
_package_refs contains ref if {
	_package_target
	_include_declaration

	some file, module in _scope
	_term_values(module.package.path) == _package_target

	ref := {
		"uri": file,
		"range": range.spanning(module.package.path),
		"kind": kind_write,
	}
}

# the imports of the package at the cursor, in any file
_package_imports contains [file, module, imp] if {
	_package_target

	some file, module in _scope
	some imp in module.imports
	_term_values(imp.path.value) == _package_target
}

# the path of the package at the cursor
_package_target := _term_values(find.package_at_position.path)
