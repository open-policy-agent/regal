# METADATA
# description: Import shadows built-in namespace
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/import-shadows-builtin
package regal.rules.imports["import-shadows-builtin"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some imp in input.imports

	imp.path.value[0].value in {"data", "input"}

	name := _significant_name(imp)
	name in ast.builtin_namespaces

	# AST quirk: while we'd ideally provide the location of the *path component*,
	# there is no location data provided for aliases. In order to be consistent,
	# we'll just provide the location of the import.
	violation := result.fail(rego.metadata.chain(), result.location(imp))
}

_significant_name(imp) := imp.alias
_significant_name(imp) := regal.last(imp.path.value).value if not imp.alias
