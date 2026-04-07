# METADATA
# description: Use raw strings for regex patterns
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/non-raw-regex-pattern
package regal.rules.idiomatic["non-raw-regex-pattern"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some term in ast.regexp.found.literals

	loc := util.to_location_object(term.location)
	row := input.regal.file.lines[loc.row - 1]

	substring(row, loc.col - 1, 1) == `"`

	violation := result.fail(rego.metadata.chain(), result.location(term))
}
