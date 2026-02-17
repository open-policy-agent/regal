# METADATA
# description: Add description of rule here!
package regal.rules.performance["equals-over-count"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some calls in ast.found.calls
	some call in calls

	# count(x) == 0
	# count(x) != 0
	# count(x) > 0
	call[0].value[0].value in {"equal", "neq", "gt"}
	call[1].type == "call"
	call[1].value[0].value[0].value == "count"
	call[2].value == 0

	violation := result.fail(rego.metadata.chain(), result.location(call))
}
