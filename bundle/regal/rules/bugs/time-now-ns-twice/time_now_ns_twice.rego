# METADATA
# description: Repeated calls to `time.now_ns`
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/time-now-ns-twice
package regal.rules.bugs["time-now-ns-twice"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule_index
	ast.function_calls[rule_index][_].name == "time.now_ns"

	some repeated in array.slice(
		[call |
			some call in ast.function_calls[rule_index]
			call.name == "time.now_ns"
		],
		1, 100,
	)

	violation := result.fail(rego.metadata.chain(), result.location(repeated))
}
