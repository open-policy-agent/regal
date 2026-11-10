package regal.lsp.util.references

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: |
#   References to the package at the cursor: where it is imported in any file, and
#   its package declarations when declarations are included. The ranges cover the last name of the package path.
# scope: document
package_refs contains ref if {
	some [file, _, imp] in _package_imports

	ref := {
		"uri": file,
		"range": range.parse(regal.last(imp.path.value).location),
		"kind": kind_read,
	}
}

# uses of the import name
package_refs contains ref if {
	some [file, module, imp] in _package_imports

	not _renaming_alias(imp)

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

# uses of the full path, like data.p.allow
package_refs contains ref if {
	_package_target

	some [file, term, static] in _full_path_uses(_package_target)
	_is_known_path(array.slice(static, 0, count(_package_target) + 1))

	ref := {
		"uri": file,
		"range": range.parse(term.location),
		"kind": kind_read,
	}
}

# the package declarations
package_refs contains ref if {
	_package_target
	_include_declaration

	some file, module in _scope
	_term_values(module.package.path) == _package_target

	ref := {
		"uri": file,
		"range": range.parse(regal.last(module.package.path).location),
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

# when renaming, an import of a package in the workspace is a rename of that package
_package_target := path if {
	_renaming

	path := _term_values(find.import_at_position.path.value)

	some module in data.workspace.parsed
	_term_values(module.package.path) == path
}

# an alias is the name of the import, not of the package, so renaming the package leaves its uses alone
_renaming_alias(imp) if {
	_renaming

	ast.imported_identifier(imp) != regal.last(imp.path.value).value
}
