# METADATA
# description: '`with` used outside test context'
package regal.rules.performance["with-outside-test-context"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some i, rule in input.rules
	not startswith(trim_prefix(rule.head.ref[0].value, "todo_"), "test_")

	_with := ast.found.expressions[ast.rule_index_strings[i]][_].with[_]
	loc := result.location(_with)

	violation := result.fail(rego.metadata.chain(), object.union(loc, {"location": {"end": {
		# only highlight 'with' itself
		"row": loc.location.row,
		"col": loc.location.col + 4,
	}}}))
}
