# METADATA
# description: Reference ignores import
package regal.rules.imports["ignored-import"]

import data.regal.ast
import data.regal.result

_import_paths contains [term.value | some term in imp.path.value] if {
	some imp in input.imports

	imp.path.value[0].value in {"data", "input"}
	count(imp.path.value) > 1
}

report contains violation if {
	ref := ast.found.refs[_][_]

	ref.value[0].type == "var"
	ref.value[0].value in {"data", "input"}

	most_specific_match := regal.last(sort([import_path |
		ref_path := [term.value | some term in ref.value]

		some import_path in _import_paths
		array.slice(ref_path, 0, count(import_path)) == import_path
	]))

	violation := result.fail(rego.metadata.chain(), object.union(
		result.location(ref),
		{"description": $"Reference ignores import of {concat(".", most_specific_match)}"},
	))
}
