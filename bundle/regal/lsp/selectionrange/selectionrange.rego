# METADATA
# description: handle textDocument/selectionRange requests
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_selectionRange
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.selectionrange
package regal.lsp.selectionrange

import data.regal.util

import data.regal.lsp.util.find
import data.regal.lsp.util.range

# METADATA
# description: as per spec, return SelectionRange[] or null
default result["response"] := null

# METADATA
# entrypoint: true
result["response"] := ranges if {
	# we don't have a array.concat_n function, and of course, no reduce
	ranges := array.concat(array.concat(package_ranges, import_ranges), rule_ranges)

	count(ranges) > 0
}

# METADATA
# description: |
#   provide selection ranges for the given positions, which may be anywhere
#   in the policy (package declaration, imports, and rules)
# scope: document

# METADATA
# description: |
#   provide selection ranges in package declaration (term, ref, line)
## this is unexpectedly the most convoluted case, as OPA's AST (and at least at this point in
## time, also RoAST) provides the worst location information for package declarations, and we
## therefore need to calculate everything but the term location ourselves
package_ranges := [item |
	some position in input.params.positions

	# regal ignore:with-outside-test-context
	pkg := data.workspace.parsed[input.params.textDocument.uri].package
	ranges := _find_ranges(pkg.path, position)

	count(ranges) > 0

	# Note: pkg.path[0] is currently just {"type": "var", "value": "data"}
	# which is useless, but an issue inherited from OPA we should get rid of in RoAST
	path_range := {"range": {
		"start": range.parse(pkg.path[1].location).start,
		"end": range.parse(regal.last(pkg.path).location).end,
	}}

	line_range := {"range": {
		"start": {"line": path_range.range.start.line, "character": 0},
		"end": path_range.range.end,
	}}

	# package locations are even worse than imports, as there isn't even a location of
	# the path itself, but only the `package` keyword and the indivdual path terms.. but OTOH,
	# copy/pasting that line is probably not too common
	item := _to_selection_range(array.concat(ranges, [path_range, line_range]))
]

# METADATA
# description: |
#   provide selection ranges in imports. these need some special handling, as
#   the user expectation is reasonably that they'd be able to expand the selection
#   to include the full import statement.. which isn't directly represented in the AST
#   node for the import (no full location, alias only a string, etc)
import_ranges := [item |
	some position in input.params.positions

	# regal ignore:with-outside-test-context
	imp := find.import_at_position with input.params.position as position
	ranges := _find_ranges(imp.path, position)

	count(ranges) > 0

	# import locations don't include the full text of the line, so we'll need to
	# improvise here some, assuming properly formatted Rego
	item := _to_selection_range(array.concat(ranges, [{"range": _estimated_import_range(imp)}]))
]

# METADATA
# description: provide selection ranges in rules
rule_ranges := [item |
	some position in input.params.positions

	# regal ignore:with-outside-test-context
	[rule, _] := find.rule_at_position with input.params.position as position
	ranges := _find_ranges(rule, position)

	count(ranges) > 0

	item := _to_selection_range(ranges)
]

# collect all ranges within the node that contain position
# ordered from most specific to least specific
_find_ranges(node, position) := array.reverse([{"range": value_range} |
	walk(node, [_, value])

	value_range := range.parse(value.location)

	range.contains_position(value_range, position)
])

# the SelectionRange object is recursive, so we need to reach for tricks here!
_to_selection_range(ranges) := json.patch(ranges[0], [patch |
	some i, r in array.slice(ranges, 1, count(ranges))

	patch := {"op": "add", "path": util.repeat("/parent", i + 1), "value": r}
])

_estimated_import_range(imp) := import_range if {
	not imp.alias

	path_range := range.parse(imp.path.location)
	import_range := {
		"start": {
			"line": path_range.start.line,
			"character": 0,
		},
		"end": {
			"line": path_range.end.line,
			"character": path_range.end.character,
		},
	}
}

_estimated_import_range(imp) := import_range if {
	imp.alias

	path_range := range.parse(imp.path.location)
	import_range := {
		"start": {
			"line": path_range.start.line,
			"character": 0,
		},
		"end": {
			"line": path_range.end.line,
			"character": (path_range.end.character + 4) + count(imp.alias), # account for " as "
		},
	}
}
