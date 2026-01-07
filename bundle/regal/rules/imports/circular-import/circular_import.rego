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

_refs contains [concat(".", [e.value | some e in r.value]), r.location] if {
	some r
	ast.found.refs[_][r].value[0].value == "data"

	ast.static_ref(r)
}

_refs contains [concat(".", [e.value | some e in imported.path.value]), imported.path.location] if {
	some imported in ast.imports

	imported.path.value[0].value == "data"
}

# METADATA
# description: collects refs from module
aggregate contains entry if {
	count(_refs) > 0

	entry := result.aggregate(rego.metadata.chain(), {"refs": _refs})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	# we need to have two or more entries in the aggregate to
	# for a circular import to be possible
	count(input.aggregate) > 1

	some g in _groups

	sorted_group := sort(g)

	location := [loc |
		some m1 in sorted_group
		some m2 in sorted_group
		some loc in _package_locations[m1][m2]
	][0]

	violation := result.fail(rego.metadata.chain(), {
		"description": $`Circular import detected in: {concat(", ", sorted_group)}`,
		"location": location,
	})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	# this rule tests for self dependencies
	some g in _groups

	count(g) == 1

	some pkg in g # this will the only package

	violation := result.fail(rego.metadata.chain(), {
		"description": $`Circular self-dependency in: {pkg}`,
		"location": [e | some e in _package_locations[pkg][pkg]][0],
	})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_package_locations[referenced_pkg][referencing_pkg] contains location if {
	some ag_pkg in input.aggregate

	some [referenced_pkg, referenced_location] in ag_pkg.aggregate_data.refs

	referencing_pkg := $"data.{concat(".", ag_pkg.aggregate_source.package_path)}"
	ref_loc := util.to_location_no_text(referenced_location)

	location := {
		"file": ag_pkg.aggregate_source.file,
		"row": ref_loc.row,
		"col": ref_loc.col,
	}
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_import_graph[pkg] contains edge if {
	some ag_pkg in input.aggregate

	pkg := $"data.{concat(".", ag_pkg.aggregate_source.package_path)}"
	edge := ag_pkg.aggregate_data.refs[_][0]
}

_reachable_index[pkg] := graph.reachable(_import_graph, {pkg}) if some pkg, _ in _import_graph

_self_reachable contains pkg if {
	some pkg, _ in _import_graph

	pkg in _reachable_index[pkg]
}

_groups contains group if {
	some pkg in _self_reachable

	# only consider packages that have edges to other packages,
	# even if only to themselves
	_import_graph[pkg] != {}

	reachable := graph.reachable(_import_graph, {pkg})

	group := {m |
		some m in reachable
		pkg in _reachable_index[m]
	}
}
