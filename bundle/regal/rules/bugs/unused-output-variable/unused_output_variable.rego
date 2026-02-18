# METADATA
# description: Unused output variable
package regal.rules.bugs["unused-output-variable"]

import data.regal.ast
import data.regal.result

# METADATA
# description: |
#   The `report` set contains unused output vars from `some` declarations. Using a
#   stricter ruleset than OPA, Regal considers an output var "unused" if it's used
#   only once in a ref, as that usage may just as well be replaced by a wildcard.
#   ```
#   some y
#   x := data.foo.bar[y]
#   # y not used later
#   ```
#   Would better be written as:
#   ```
#   some x in data.foo.bar
#   ```
report contains violation if {
	some rule_index
	var_refs := ast.found.vars[rule_index].ref

	count(var_refs) == 1

	some var in var_refs

	not ast.is_wildcard(var)
	not ast.var_in_head(input.rules[to_number(rule_index)].head, var.value)
	not ast.var_in_call(ast.function_calls, rule_index, var.value)
	not _ref_base_vars[rule_index][var.value]
	not _comprehension_term_vars[rule_index][var.value]

	# this is by far the most expensive condition to check, so only do
	# so when all other conditions apply
	ast.is_output_var(input.rules[to_number(rule_index)], var)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}

# "a" in "a[foo]", and not foo
_ref_base_vars[rule_index][term.value] contains term if {
	some rule_index
	term := ast.found.refs[rule_index][_].value[0]
}

_comprehension_term_vars[rule_index] contains var.value if {
	some rule_index, comprehensions in ast.found.comprehensions
	some comprehension in comprehensions
	some var in ast.find_term_vars(object.remove(comprehension.value, ["body"]))
}
