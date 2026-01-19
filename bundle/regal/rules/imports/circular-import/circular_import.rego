# METADATA
# description: Circular import
# related_resources:
# - description: documentation
#   ref: https://www.openpolicyagent.org/projects/regal/rules/imports/circular-import
# schemas:
# - input: schema.regal.ast
package regal.rules.imports["circular-import"]

import data.regal.ast
import data.regal.result
import data.regal.util

_refs[ref] contains r.location if {
	some r
	ast.found.refs[_][r].value[0].value == "data"
	ast.static_ref(r)

	ref := concat(".", [e.value | some e in r.value])
}

_refs[ref] contains imported.path.location if {
	some imported in ast.imports

	imported.path.value[0].value == "data"

	ref := concat(".", [e.value | some e in imported.path.value])
}

# METADATA
# description: collects refs from module, if any
aggregate contains {"package_name": ast.package_name_full, "refs": _refs} if count(_refs) > 0

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	# 2+ files required in the aggregated set for a circular import to be possible
	count(_aggregated) > 1

	some g in _groups

	count(g) > 1

	sorted_group := sort(g)

	[file, referenced_location] := [file_loc |
		some m1 in sorted_group
		some m2 in sorted_group
		file_loc := _package_locations[m1][m2]
	][0]

	violation := result.fail(rego.metadata.chain(), {
		"description": $"Circular import detected in: {concat(", ", sorted_group)}",
		"location": object.union(util.to_location_no_text(referenced_location), {"file": file}),
	})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	# this rule tests for self dependencies
	some g in _groups

	count(g) == 1

	some pkg in g # this will be the only package

	[file, referenced_location] := _package_locations[pkg][pkg]

	violation := result.fail(rego.metadata.chain(), {
		"description": $"Circular self-dependency in: {pkg}",
		"location": object.union(util.to_location_no_text(referenced_location), {"file": file}),
	})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_package_locations[referenced_pkg][pkg.package_name] := [file, util.any_set_item(referenced_locations)] if {
	some file, pkg in _aggregated
	some referenced_pkg, referenced_locations in pkg.refs
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_import_graph[pkg.package_name] contains edge if {
	some pkg in _aggregated
	some edge, _ in pkg.refs
}

_reachable_index[pkg] := graph.reachable(_import_graph, {pkg}) if some pkg, _ in _import_graph

_groups contains group if {
	some pkg, _ in _import_graph

	pkg in _reachable_index[pkg] # self-reachable

	# only consider packages that have edges to other packages,
	# even if only to themselves
	count(_import_graph[pkg]) > 0

	group := {m |
		some m in graph.reachable(_import_graph, {pkg})
		pkg in _reachable_index[m]
	}
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_aggregated[file] := agg if {
	# we know that there is only one aggregate of this type per file,
	# so we can simplify things some for our callers
	some file
	agg := input.aggregates_internal[file]["imports/circular-import"][_]
}
