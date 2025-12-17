# METADATA
# description: Unresolved Reference
package regal.rules.imports["unresolved-reference"]

import data.regal.ast
import data.regal.config
import data.regal.result

# METADATA
# description: collects exported and full of used refs from each module
aggregate contains entry if {
	exported := {rule |
		some rule in object.keys(ast.rule_head_locations)
		not rule in unexported_rules
	}

	entry := {
		"exported_rules": exported,
		"expanded_refs": _all_full_path_refs,
		"prefix_set": {["data"]} | {prefix_path |
			some rule_name in exported
			rule_path := split(rule_name, ".")
			some i in numbers.range(2, count(rule_path))
			prefix_path := array.slice(rule_path, 0, i)
		},
	}
}

default _excepted_export_patterns := {"**.test_*"}

_excepted_export_patterns := config.rules.imports["unresolved-reference"].excepted_export_patterns

# METADATA
# description: removes rules that are ignored in the config file
unexported_rules contains rule_name if {
	some rule_name, _ in ast.rule_head_locations
	some exception in _excepted_export_patterns
	glob.match(exception, [], rule_name)
}

# an import is shadowed if it shares name with a rule
_shadowed_imports contains rule_name if {
	some rule_name in ast.rule_names
	ast.resolved_imports[rule_name]
}

# an import is shadowed if it shares name with a variable (or function argument)
_shadowed_imports contains var_name if {
	var_name := ast.found.vars[_][_][_].value
	ast.resolved_imports[var_name]
}

_refs contains ref if {
	terms := ast.found.refs[_][_].value
	terms[0].value != "input"

	name := ast.ref_static_to_string(terms)

	not name in ast.builtin_names
	not name in ast.rule_and_function_names
	not name in unexported_rules

	not terms[0].value in _shadowed_imports

	row := to_number(regex.replace(terms[0].location, `^(\d+):.*`, "$1"))
	ref := {
		"name": name,
		"text": input.regal.file.lines[row - 1],
		"location": terms[0].location,
	}
}

_all_full_path_refs[ref.name] contains [ref.location, ref.text] if {
	some ref in _refs
	startswith(ref.name, "data.")
}

_all_full_path_refs[expanded] contains [ref.location, ref.text] if {
	some ref in _refs

	ref_root := regex.replace(ref.name, `^([^\.]+)\..*`, "$1") # anything before the first ".", like "bar" in "foo.bar"
	resolved := concat(".", ast.resolved_imports[ref_root]) #    resolve that root, e.g. "data.regal.foo"
	expanded := regex.replace(ref.name, `^([^\.]+)`, resolved) # add back the suffix, e.g. "data.regal.foo.bar"
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	all_exports := {export | export := _aggregates[_][_].exported_rules[_]}
	prefix_set := {prefix | prefix := _aggregates[_][_].prefix_set[_]}

	some file
	entry := _aggregates[file][_]
	some name, refs in entry.expanded_refs

	# ignore everything from the first "[" in the ref name. E.g. foo.bar[0].baz becomes foo.bar
	ref_name := regex.replace(name, `^([^\[]+)\[.*`, "$1")
	ref_path := split(ref_name, ".")

	# a reference is considered resolved with respect to a rule if
	# 1: it is the prefix of a rule
	# 2: it indexes into a rule - we do not consider the possible data
	# 3: the reference is ignored in the config
	not ref_path in prefix_set
	not _is_resolved_ref(ref_path, all_exports)
	not _is_excepted(ref_name)

	some ref in refs

	violation := result.fail(rego.metadata.chain(), _to_location_object(ref[0], ref[1], file))
}

_is_excepted(ref_full_name) if {
	some exception in config.rules.imports["unresolved-reference"]["except-paths"]
	glob.match(exception, [], ref_full_name)
}

_is_resolved_ref(ref_full_path, all_exports) if {
	some i in numbers.range(1, count(ref_full_path))
	path_prefix := concat(".", array.slice(ref_full_path, 0, i))

	path_prefix in all_exports
}

# like util.to_location_object, but with text and file passed in
# as we don't have access to the usual input.regal.file attributes
# in the context of reporting aggregated data
_to_location_object(loc, text, file) := {"location": {
	"file": file,
	"row": row,
	"col": col,
	"text": text,
	"end": {
		"row": row,
		"col": end_col,
	},
}} if {
	vals := split(loc, ":")

	row := to_number(vals[0])
	col := to_number(vals[1])

	from_col := substring(text, col - 1, -1)
	ref_text := substring(from_col, 0, indexof(from_col, " "))

	end_col := col + count(ref_text)
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_aggregates[file] := input.aggregates_internal[file]["imports/unresolved-reference"] if some file
