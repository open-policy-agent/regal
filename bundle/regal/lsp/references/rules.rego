package regal.lsp.references

import future.keywords.or

import data.regal.ast
import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: |
#   References to the rule or function at the cursor, by bare name or full path.
#   When declarations are included, its heads in any file of the package are returned too.
_rule_refs contains ref if {
	_rule_target

	name := regal.last(_rule_target)

	some file, module in data.workspace.parsed
	_rule_path(module.package.path, name) == _rule_target

	some rule in module.rules
	not _shadows_package_rule(rule, name)

	some node in ["head", "body", "else"]
	walk(rule[node], [_, var])

	var.value == name
	var.type == "var"
	_include_declaration or var != rule.head.ref[0]

	ref := {
		"uri": file,
		"range": range.parse(var.location),
	}
}

# the full path can be used from anywhere, the name is the term following the pkg
_rule_refs contains ref if {
	_rule_target

	some file, module in data.workspace.parsed
	some rule in module.rules
	some node in ["head", "body", "else"]

	walk(rule[node], [_, value])

	value.type == "ref"

	static := [term.value | some term in ast.static_prefix(value.value)]
	array.slice(static, 0, count(_rule_target)) == _rule_target

	ref := {
		"uri": file,
		"range": range.parse(value.value[count(_rule_target) - 1].location),
	}
}

# a package can be used through its import, the name is the term following the import
_rule_refs contains ref if {
	_rule_target

	some file, module in data.workspace.parsed
	some imp in module.imports

	imp_path := _import_path(imp.path.value)
	count(imp_path) < count(_rule_target)
	array.slice(_rule_target, 0, count(imp_path)) == imp_path
	tail := array.slice(_rule_target, count(imp_path), count(_rule_target))

	id := ast.imported_identifier(imp)

	some rule in module.rules
	not _shadows_package_rule(rule, id)

	some node in ["head", "body", "else"]
	walk(rule[node], [_, value])

	value.type == "ref"
	value.value[0].type == "var"
	value.value[0].value == id
	[term.value | some term in array.slice(value.value, 1, count(tail) + 1); term.type == "string"] == tail

	ref := {
		"uri": file,
		"range": range.parse(value.value[count(tail)].location),
	}
}

# on a var
_rule_target := path if {
	[var, rule_index] := find.var_at_position

	not _shadows_package_rule(_module.rules[rule_index], var.value)

	path := _rule_path(_module.package.path, var.value)
	_is_known_path(path)
}

# on a full path
_rule_target := prefix if {
	not find.var_at_position

	[rule, _] := find.rule_at_position

	walk(rule, [_, ref])

	ref.type == "ref"
	ref.value[0].value == "data"

	some i, term in ref.value
	i > 0
	range.contains_position(range.parse(term.location), input.params.position)

	prefix := [t.value | some t in array.slice(ref.value, 0, i + 1)]
	_is_known_path(prefix)
}

# on a use through an import
_rule_target := path if {
	not find.var_at_position

	[rule, _] := find.rule_at_position

	walk(rule, [_, ref])

	ref.type == "ref"
	ref.value[0].type == "var"

	some i, term in ref.value
	i > 0
	range.contains_position(range.parse(term.location), input.params.position)

	some imp in _module.imports
	ast.imported_identifier(imp) == ref.value[0].value
	not _shadows_package_rule(rule, ref.value[0].value)

	terms := array.slice(ref.value, 1, i + 1)
	tail := [t.value | some t in terms; t.type == "string"]
	count(tail) == count(terms)

	path := array.concat(_import_path(imp.path.value), tail)
	_is_known_path(path)
}

# true if a rule with this full path is defined in the workspace
_is_known_path(path) if {
	name := regal.last(path)

	some module in data.workspace.parsed
	_rule_path(module.package.path, name) == path

	some rule in module.rules
	rule.head.ref[0].value == name
}

_rule_path(package_path, name) := array.flatten([
	[term.value | some term in package_path],
	name,
])

_import_path(terms) := [term.value | some term in terms]

# true if an argument or a := variable in the rule has the name of a rule in the package
_shadows_package_rule(rule, package_rule) if {
	some arg in rule.head.args
	ast.has_named_var(arg, package_rule)
}

_shadows_package_rule(rule, package_rule) if {
	some expr in rule.body

	expr.terms[0].value[0].value == "assign"
	ast.has_named_var(expr.terms[1], package_rule)
}
