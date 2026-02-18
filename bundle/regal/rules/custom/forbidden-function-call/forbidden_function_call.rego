# METADATA
# description: Forbidden function call
package regal.rules.custom["forbidden-function-call"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	forbidden := util.to_set(config.rules.custom["forbidden-function-call"]["forbidden-functions"])

	# avoid traversal if no forbidden function is called
	util.intersects(forbidden, ast.builtin_functions_called)

	ref := ast.found.calls[_][_]

	ast.ref_to_string(ref[0].value) in forbidden

	violation := result.fail(rego.metadata.chain(), result.location(ref))
}
