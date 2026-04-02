# METADATA
# description: Import shadows another import
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/import-shadows-import
package regal.rules.imports["import-shadows-import"]

import data.regal.capabilities
import data.regal.result

# METADATA
# description: Since OPA 1.0, import-shadows-import enabled only when provided a v0 policy,
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

# regular import
_ident(imported) := regal.last(imported.path.value).value if not imported.alias

# aliased import
_ident(imported) := imported.alias

_identifiers := [_ident(imported) | some imported in input.imports]

report contains violation if {
	some i, identifier in _identifiers

	identifier in array.slice(_identifiers, 0, i)

	violation := result.fail(rego.metadata.chain(), result.location(input.imports[i].path))
}
