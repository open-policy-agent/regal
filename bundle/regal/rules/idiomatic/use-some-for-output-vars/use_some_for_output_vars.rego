# METADATA
# description: Use `some` to declare output variables
package regal.rules.idiomatic["use-some-for-output-vars"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some rule_index, term
	ast.found.vars[rule_index].ref[term]

	not startswith(term.value, "$")
	not term.value in ast.imported_identifiers
	not term.value in ast.rule_names

	rule := input.rules[to_number(rule_index)]

	not ast.is_in_local_scope(rule, term.location, term.value)

	walk(rule, [path, term.location])

	not _var_in_ref_head_declared(path, rule_index, term.value)
	not _var_in_comprehension_body(path, term.value, rule)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}

_var_in_comprehension_body(path, value, rule) if {
	some parent_path in array.reverse(util.all_paths(path))

	node := object.get(rule, parent_path, {})

	node.type in {"arraycomprehension", "objectcomprehension", "setcomprehension"}

	some v in ast.find_vars(node.value.body)
	v.type == "var"
	v.value == value
}

_var_in_ref_head_declared(path, rule_index, value) if {
	path[0] == "head"
	path[1] == "ref"
	ast.found.vars[rule_index]["some"][_].value == value
}
