# METADATA
# description: Prefer value in rule head
package regal.rules.custom["prefer-value-in-head"]

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	some rule in input.rules

	terms := regal.last(rule.body).terms

	terms[0].value[0].type == "var"
	terms[0].value[0].value in {"eq", "assign"}
	terms[1].type == "var"
	terms[1].value == [rule.head[attribute].value |
		some attribute in ["value", "key"]

		rule.head[attribute].type == "var"
	][0]

	not _scalar_fail(terms[2].type, _scalar_types)
	not _excepted_var_name(terms[1].value)

	violation := result.fail(rego.metadata.chain(), result.location(terms[2]))
}

_scalar_fail(term_type, scalar_types) if {
	config.rules.custom["prefer-value-in-head"]["only-scalars"] == true
	not term_type in scalar_types
}

_excepted_var_name(name) if name in config.rules.custom["prefer-value-in-head"]["except-var-names"]

_scalar_types contains type if some type in ast.scalar_types
_scalar_types contains "templatestring" if config.rules.custom["prefer-value-in-head"]["include-interpolated"] == true
