# METADATA
# description: Import declared after rule
package regal.rules.imports["import-after-rule"]

import data.regal.result
import data.regal.util

report contains violation if {
	first_rule_row := to_number(util.substring_to(input.rules[0].location, 0, ":"))

	some imp in input.imports

	to_number(util.substring_to(imp.location, 0, ":")) > first_rule_row

	violation := result.fail(rego.metadata.chain(), result.location(imp))
}
