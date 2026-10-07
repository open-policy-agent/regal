# METADATA
# description: Rule defined unconditionally alongside default definition
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/impossible-default
package regal.rules.bugs["impossible-default"]

import future.keywords.or

import data.regal.ast
import data.regal.result

report contains violation if {
	some default_rule in input.rules
	default_rule.default == true

	_has_unconditional_definition(default_rule)

	violation := result.fail(rego.metadata.chain(), result.location(default_rule))
}

_has_unconditional_definition(default_rule) if {
	default_target := _rule_target(default_rule)

	some rule in input.rules
	not rule.default
	not rule.body

	_rule_target(rule) == default_target
	_unconditional(rule)
}

_rule_target(rule) := [ast.ref_to_string(rule.head.ref), count(object.get(rule.head, "args", []))] if {
	rule.head.value
	not _dynamic_ref(rule.head.ref)
}

_dynamic_ref(ref) if array.slice(ref, 1, 100)[_].type in {"call", "var", "ref", "templatestring"}

_unconditional(rule) if {
	not rule["else"]
	ast.is_constant(rule.head.value)

	not rule.head.args or {
		every arg in rule.head.args {
			arg.type == "var"
		}

		count({arg.value | some arg in rule.head.args}) == count(rule.head.args)
	}
}
