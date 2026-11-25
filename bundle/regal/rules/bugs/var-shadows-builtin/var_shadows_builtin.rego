# METADATA
# description: Variable name shadows built-in
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/var-shadows-builtin
package regal.rules.bugs["var-shadows-builtin"]

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	var := ast.found.vars[_][_][_]

	var.value in ast.builtin_namespaces
	not var.value in _except_names

	violation := result.fail(rego.metadata.chain(), result.location(var))
}

_except_names contains name if some name in config.rules.bugs["var-shadows-builtin"]["except-names"]
