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

	some var in _named_vars(rule, name)
	not _is_local(rule, var)
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

	some node in ["head", "body", "else"]
	walk(rule[node], [_, value])

	value.type == "ref"
	value.value[0].value == id
	not _is_local(rule, value.value[0])
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

	some var in _uses_of(rule, id)
	not _is_local(rule, var)

	ref := {
		"uri": file,
		"range": range.parse(var.location),
		"kind": kind_read,
	}
}

# on a var
_rule_target := path if {
	[var, rule_index] := find.var_at_position

	not _is_local(_module.rules[rule_index], var)

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
	not _is_local(rule, terms[0])

	imp_path := _term_values(imp.path.value)
	path := array.slice(_through_import(imp_path, terms), 0, count(imp_path) + i)
	_is_known_path(path)
}

# on a use of a rule that is imported directly, like allow after import data.pkg.allow
_rule_target := path if {
	[var, rule_index] := find.var_at_position

	not _is_local(_module.rules[rule_index], var)
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

# true if the var is bound by a local declaration in a scope that it is in. The declaration can be an argument,
# or a variable declared by :=, some or every. A local var hides any rule, import or package of the same name.
_is_local(rule, var) if {
	var.type == "var"
	some arg in rule.head.args
	ast.has_named_var(arg, var.value)
}

_is_local(rule, var) if {
	walk(rule, [use_path, var])
	walk(rule, [declaration_path, expr])

	not expr.terms.domain
	some declared in _declared_vars(expr.terms)
	ast.has_named_var(declared, var.value)

	_in_scope(use_path, array.slice(declaration_path, 0, count(declaration_path) - 2))
}

_is_local(rule, var) if {
	walk(rule, [use_path, var])
	walk(rule, [declaration_path, expr])

	expr.terms.domain
	some declared in _declared_vars(expr.terms)
	ast.has_named_var(declared, var.value)

	_in_scope(use_path, array.flatten([declaration_path, "terms"]))
}

_in_scope(use_path, scope) if {
	array.slice(use_path, 0, count(scope)) == scope
	not use_path[count(scope)] in {"else", "domain"}
}

# the variables declared by :=, every and some, not including the domain they iterate
_declared_vars(terms) := [lhs] if [lhs, _] := ast.assignment_terms(terms)

_declared_vars(terms) := [var | some key in ["key", "value"]; var := terms[key]] if terms.domain

_declared_vars(terms) := array.flatten([_declared_by(symbol) | some symbol in terms.symbols]) if terms.symbols

_declared_by(symbol) := [symbol] if symbol.type != "call"

_declared_by(symbol) := array.slice(symbol.value, 1, count(symbol.value) - 1) if symbol.type == "call"
