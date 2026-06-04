# METADATA
# description: Unresolved Reference
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/unresolved-reference
package regal.rules.imports["unresolved-reference"]

import future.keywords.not

import data.regal.aggregated
import data.regal.ast
import data.regal.config
import data.regal.result

# METADATA
# description: collects exported and full of used refs from each module
aggregate contains {"expanded_refs": _all_full_path_refs}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	rule_tree := aggregated.rule_tree

	some name, files in _aggregated_refs

	# ignore everything from the first "[" in the ref name. E.g. foo.bar[0].baz becomes foo.bar
	ref_name := regex.replace(name, `^([^\[]+)\[.*`, "$1")
	ref_path := split(ref_name, ".")

	# a reference is considered resolved with respect to a rule if
	# it indexes into a rule, or is the prefix of a rule, or the
	# reference is ignored in the config
	not _is_resolved_ref(ref_path, rule_tree)
	not {
		some exception in config.rules.imports["unresolved-reference"]["except-paths"]
		glob.match(exception, [], ref_name)
	}

	some file, locations in files
	some location in locations

	violation := result.fail(rego.metadata.chain(), aggregated.location_object(location, file))
}

default _excepted_export_patterns := {"**.test_*"}

_excepted_export_patterns := config.rules.imports["unresolved-reference"].excepted_export_patterns

# an import is shadowed if it shares name with a rule
_shadowed_imports contains rule_name if {
	some rule_name in ast.rule_names
	_ = ast.resolved_imports[rule_name]
}

# an import is shadowed if it shares name with a variable (or function argument)
_shadowed_imports contains var_name if {
	var_name := ast.found.vars[_][_][_].value
	_ = ast.resolved_imports[var_name]
}

_refs[name] contains terms[0].location if {
	some rule_index
	terms := ast.found.refs[rule_index][_].value
	head := terms[0].value
	head != "input"

	name := ast.ref_static_to_string(terms)

	not name in ast.builtin_names
	not name in ast.rule_and_function_names
	not head in _shadowed_imports
}

_all_full_path_refs[name] contains location if {
	some name, locations in _refs

	startswith(name, "data.")

	some location in locations
}

_all_full_path_refs[expanded] contains location if {
	some name, locations in _refs

	ref_root := regex.replace(name, `^([^\.]+)\..*`, "$1") # anything before the first ".", like "bar" in "foo.bar"
	resolved := concat(".", ast.resolved_imports[ref_root]) #    resolve that root, e.g. "data.regal.foo"
	expanded := regex.replace(name, `^([^\.]+)`, resolved) # add back the suffix, e.g. "data.regal.foo.bar"

	some location in locations
}

_aggregated_refs[name][file] := locations if {
	some file
	entry := _aggregates[file][_]

	some name, locations in entry.expanded_refs
}

_is_resolved_ref(ref_path, rule_tree) if is_object(object.get(rule_tree, ref_path, false))

_is_resolved_ref(ref_path, rule_tree) if {
	some i in numbers.range(1, count(ref_path))

	object.get(rule_tree, array.slice(ref_path, 0, i), false) == {} # regal ignore:superfluous-object-get
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_aggregates[file] := input.aggregates_internal[file]["imports/unresolved-reference"] if some file
