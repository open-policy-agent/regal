# METADATA
# description: Unnecessary use of `some`
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/unnecessary-some
package regal.rules.style["unnecessary-some"]

import data.regal.ast
import data.regal.result

report contains violation if {
	# No need to traverse rules here if we're not importing `in`
	ast.imports_keyword(ast.imports, "in")

	symbols := ast.found.symbols[_][_]

	symbols[0].type == "call"
	symbols[0].value[0].type == "ref"

	_some_is_unnecessary(symbols[0].value, ast.scalar_types)

	violation := result.fail(rego.metadata.chain(), result.location(symbols))
}

_some_is_unnecessary(symbol, scalar_types) if {
	symbol[0].value[0].value == "internal"
	symbol[0].value[1].value == "member_2"
	symbol[1].type in scalar_types
}

_some_is_unnecessary(symbol, scalar_types) if {
	symbol[0].value[0].value == "internal"
	symbol[0].value[1].value == "member_3"
	symbol[1].type in scalar_types
	symbol[2].type in scalar_types
}
