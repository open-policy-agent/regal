# METADATA
# description: |
#   common collections of *aggregated* data, for use in aggregate_report rules
# schemas:
#   - input: schema.regal.aggregate
package regal.aggregated

import data.regal.util

# METADATA
# description: |
#   a tree representing all rules in the linted files, e.g.:
#   {
#     "data": {
#       "package1": {
#         "rule1": {},
#         "nested": {
#           "rule2": {},
#         },
#       },
#       "package2": {
#         "rule3": {},
#       },
#     }
#   }
rule_tree := object.union_n([m | m := _aggregates[_][_].rule_tree])

# METADATA
# description: |
#   a set containing all package paths from the linted files
all_package_paths := {path | path := _aggregates[_][_].package_path}

# METADATA
# description: |
#   like util.to_location_object, but with text and file passed in
#   as we don't have access to the usual input.regal.file attributes
#   in the context of reporting aggregated data
location_object(loc, file) := {"location": {
	"file": file,
	"row": row,
	"col": col,
	"text": text,
	"end": {
		"row": row,
		"col": col + count(ref_text),
	},
}} if {
	pos := split(loc, ":")
	row := to_number(pos[0])
	col := to_number(pos[1])

	text := util.any_set_item(input.aggregates_internal[file].common).lines[row - 1]

	from_col := substring(text, col - 1, -1)
	ref_text := substring(from_col, 0, indexof(from_col, " "))
}

_aggregates[file] := input.aggregates_internal[file].common if some file
