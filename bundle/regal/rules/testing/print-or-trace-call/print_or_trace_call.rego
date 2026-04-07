# METADATA
# description: Call to print or trace function
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/print-or-trace-call
package regal.rules.testing["print-or-trace-call"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	# skip iteration of refs if no print or trace calls are registered
	util.intersects(ast.builtin_functions_called, {"print", "trace"})

	ref := ast.found.calls[_][_][0]

	ref.value[0].type == "var"
	ref.value[0].value in {"print", "trace"}

	violation := result.fail(rego.metadata.chain(), result.location(ref))
}
