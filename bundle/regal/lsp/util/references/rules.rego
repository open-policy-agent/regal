package regal.lsp.util.references

import future.keywords.or

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range
import data.regal.util

# METADATA
# description: |
#   References to the rule or function at the cursor, by bare name, full path or through an import.
#   When declarations are included, its heads in any file of the package are returned too.
# scope: document
rule_refs contains ref if {
	_rule_target

	name := regal.last(_rule_target)

	some file, module in _scope
	_rule_path(module.package.path, name) == _rule_target

	some rule in module.rules
	not _declares_local(rule, name)

	some var in _named_vars(rule, name)
	_include_declaration or var != rule.head.ref[0]

	var == rule.head.ref[0] or not _imports_name(object.get(module, "imports", []), name)

	ref := {
		"uri": file,
		"range": range.parse(var.location),
		"kind": _kind_write_if(var == rule.head.ref[0]),
	}
}

# the full path can be used from anywhere, the name is the term following the pkg
rule_refs contains ref if {
	_rule_target

	some [file, term, _] in _full_path_uses(_rule_target)

	ref := {
		"uri": file,
		"range": range.parse(term.location),
		"kind": kind_read,
	}
}

# a rule can be used through the import of its package, the name is the term following the import
rule_refs contains ref if {
	_rule_target

	some file, module in _scope
	some imp in module.imports

	# skips the rules of modules that cannot use the target through this import
	imp_path := _term_values(imp.path.value)
	count(imp_path) < count(_rule_target)
	array.slice(_rule_target, 0, count(imp_path)) == imp_path

	id := ast.imported_identifier(imp)

	some rule in module.rules
	not _declares_local(rule, id)

	some node in ["head", "body", "else"]
	walk(rule[node], [_, value])

	value.type == "ref"
	value.value[0].value == id
	array.slice(_through_import(imp_path, value.value), 0, count(_rule_target)) == _rule_target

	ref := {
		"uri": file,
		"range": range.parse(value.value[count(_rule_target) - count(imp_path)].location),
		"kind": kind_read,
	}
}

# a rule can be imported directly, the name is the last term of the import path
rule_refs contains ref if {
	_rule_target

	some file, module in _scope
	some imp in module.imports
	_term_values(imp.path.value) == _rule_target

	ref := {
		"uri": file,
		"range": range.parse(regal.last(imp.path.value).location),
		"kind": kind_read,
	}
}

rule_refs contains ref if {
	_rule_target

	some file, module in _scope
	some imp in module.imports
	_term_values(imp.path.value) == _rule_target
	not _renaming_alias(imp)

	id := ast.imported_identifier(imp)

	some rule in module.rules
	not _declares_local(rule, id)

	some var in _named_vars(rule, id)
	var != rule.head.ref[0]

	ref := {
		"uri": file,
		"range": range.parse(var.location),
		"kind": kind_read,
	}
}

# on a var
_rule_target := path if {
	[var, rule_index] := find.var_at_position

	not _declares_local(_module.rules[rule_index], var.value)

	var == _module.rules[rule_index].head.ref[0] or not _imports_name(object.get(_module, "imports", []), var.value)

	path := _rule_path(_module.package.path, var.value)
	_is_known_path(path)
}

# on a full path
_rule_target := prefix if {
	some [_, terms, i] in _refs_at_cursor

	prefix := _term_values(array.slice(terms, 0, i + 1))
	_is_known_path(prefix)
}

# on a use through an import
_rule_target := path if {
	some [rule, terms, i] in _refs_at_cursor

	some imp in _module.imports
	ast.imported_identifier(imp) == terms[0].value
	not _declares_local(rule, terms[0].value)

	imp_path := _term_values(imp.path.value)
	path := array.slice(_through_import(imp_path, terms), 0, count(imp_path) + i)
	_is_known_path(path)
}

# on a use of a rule that is imported directly, like allow after import data.pkg.allow
_rule_target := path if {
	[var, rule_index] := find.var_at_position

	not _declares_local(_module.rules[rule_index], var.value)
	var != _module.rules[rule_index].head.ref[0]

	some imp in _module.imports
	ast.imported_identifier(imp) == var.value

	path := _term_values(imp.path.value)
	_is_known_path(path)
}

_imports_name(imports, name) if {
	some imp in imports
	ast.imported_identifier(imp) == name
}

# refs with a name under the cursor, as [rule, ref terms, index of the name]
_refs_at_cursor contains [rule, ref.value, i] if {
	not find.var_at_position

	[rule, _] := find.rule_at_position

	walk(rule, [_, ref])

	ref.type == "ref"

	some i, term in ref.value
	i > 0
	range.contains_position(range.parse(term.location), input.params.position)
}

# true if a rule with this full path is defined in the workspace
_is_known_path(path) if {
	name := regal.last(path)

	some module in data.workspace.parsed
	_rule_path(module.package.path, name) == path

	some rule in module.rules
	rule.head.ref[0].value == name
}

_full_path_uses(path) := {[file, value.value[count(path) - 1], static] |
	some file, module in _scope
	some rule in module.rules
	some node in ["head", "body", "else"]

	walk(rule[node], [_, value])

	value.type == "ref"

	static := _term_values(ast.static_prefix(value.value))
	array.slice(static, 0, count(path)) == path
}

# the full path of a rule, as the values of its package path followed by its name
_rule_path(package_path, name) := array.flatten([_term_values(package_path), name])

_term_values(terms) := [term.value | some term in terms]

# the full path of a ref starting with an import name, as far as the ref is static
_through_import(imp_path, ref_terms) := array.concat(imp_path, [term.value |
	some term in util.rest(ast.static_prefix(ref_terms))
])

# true if an argument or a := variable in the rule has this name, which hides a rule or import of that name
_declares_local(rule, name) if {
	some arg in rule.head.args
	ast.has_named_var(arg, name)
}

_declares_local(rule, name) if {
	some expr in rule.body

	[lhs, _] := ast.assignment_terms(expr.terms)
	ast.has_named_var(lhs, name)
}
