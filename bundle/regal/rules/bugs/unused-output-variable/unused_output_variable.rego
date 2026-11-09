# METADATA
# description: Unused output variable
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/unused-output-variable
package regal.rules.bugs["unused-output-variable"]

import future.keywords.and
import future.keywords.not

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

	some term in var_refs

	not {
		term.type == "var"
		startswith(term.value, "$")
	}
	not ast.var_in_head(input.rules[rule_index].head, term.value)
	not ast.var_in_call(ast.function_calls, rule_index, term.value)
	not {
		ast.found.refs[rule_index][_].value[0].value == term.value # "a" in "a[foo]", and not foo
	}
	not {
		some comprehension in ast.found.comprehensions[rule_index]
		some declared in ["term", "key", "value"]
		some var in ast.find_term_vars(comprehension.value[declared])

		var.value == term.value
	}

	# this is by far the most expensive condition to check, so only do
	# so when all other conditions apply
	ast.is_output_var(rule_index, term)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}
