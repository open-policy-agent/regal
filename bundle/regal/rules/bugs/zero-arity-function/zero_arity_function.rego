# METADATA
# description: Avoid functions without args
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/zero-arity-function
package regal.rules.bugs["zero-arity-function"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	# notably, not ast.functions, as zero-arity functions are treated
	# as regular rules (i.e. they have no `args` key in the head)
	head := ast.rules[_].head

	text := util.to_location_object(head.location).text

	regex.match(`^[a-zA-z1-9_\.\[\]"]+\(\)`, text)

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(head.ref))
}
