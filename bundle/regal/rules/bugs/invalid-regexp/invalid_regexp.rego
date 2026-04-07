# METADATA
# description: Invalid regular expression
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/invalid-regexp
package regal.rules.bugs["invalid-regexp"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some term in ast.regexp.found.literals

	not regex.is_valid(term.value)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}
