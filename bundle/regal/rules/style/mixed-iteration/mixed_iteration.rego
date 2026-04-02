# METADATA
# description: Mixed iteration style
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/mixed-iteration
package regal.rules.style["mixed-iteration"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule_index, symbol
	ast.found.symbols[rule_index][symbol][0].type == "call"

	last := regal.last(symbol[0].value)
	last.type == "ref"

	some i, term in last.value

	i > 0
	term.type == "var"
	ast.is_output_var(input.rules[to_number(rule_index)], term)

	violation := result.fail(rego.metadata.chain(), result.location(last))
}
