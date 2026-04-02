# METADATA
# description: Function argument used for return value
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/function-arg-return
package regal.rules.style["function-arg-return"]

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	included_functions := ast.all_function_names - _excluded_functions

	some fun
	ast.function_calls[_][fun].name in included_functions

	count(fun.args) > count(ast.all_functions[fun.name].decl.args)

	violation := result.fail(rego.metadata.chain(), result.location(regal.last(fun.args)))
}

_excluded_functions contains "print"
_excluded_functions contains name if some name in config.rules.style["function-arg-return"]["except-functions"]
