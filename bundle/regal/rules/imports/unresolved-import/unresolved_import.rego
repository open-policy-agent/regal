# METADATA
# description: Unresolved import
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/unresolved-import
package regal.rules.imports["unresolved-import"]

import data.regal.aggregated
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	rule_tree := aggregated.rule_tree

	some file, entries in _aggregates
	some entry in entries
	some [path, location] in entry.imports

	not path in _except_imports
	object.get(rule_tree.data, path, false) == false

	# cheap operation failed — need to check wildcards here to account
	# for map generating / general ref head rules
	not _wildcard_match(path, _except_imports)
	not _is_resolved_ref(path, rule_tree.data)

	violation := result.fail(rego.metadata.chain(), aggregated.location_object(location, file))
}

# the package part will always be included exported refs
# but if we have a rule like foo.bar.baz
# we'll want to include both foo.bar and foo.bar.baz
_to_paths(pkg_path, ref) := util.all_paths(_to_path(pkg_path, ref)) if count(ref) < 3
_to_paths(pkg_path, ref) := [_to_path(pkg_path, terms) | some terms in util.all_paths(ref)] if count(ref) > 2

_to_path(pkg_path, terms) := array.flatten([pkg_path, terms[0].value, [_to_string(term) |
	some term in array.slice(terms, 1, 100)
]])

_to_string(term) := term.value if term.type == "string"
_to_string(term) := "**" if term.type == "var"

_is_resolved_ref(ref_path, rule_tree) if {
	some i in numbers.range(1, count(ref_path))

	object.get(rule_tree, array.slice(ref_path, 0, i), false) == {} # regal ignore:superfluous-object-get
}

_except_imports contains split(trim_prefix(str, "data."), ".") if {
	some str in config.rules.imports["unresolved-import"]["except-imports"]
}

_wildcard_match(imp_path, except_imports) if {
	some except in except_imports
	path := concat(".", except)
	contains(path, "*")

	# note that we are quite forgiving here, as we'll match the
	# shortest path component containing a wildcard at the end..
	# we may want to make this more strict later, but as this is
	# a new rule with a potentially high impact, let's start like
	# this and then decide if we want to be more strict later, and
	# perhaps offer that as a "strict" option
	glob.match(path, [], concat(".", imp_path))
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_aggregates[file] := input.aggregates_internal[file].common if some file
