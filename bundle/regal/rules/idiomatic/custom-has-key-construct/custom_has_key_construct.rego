# METADATA
# description: Custom function may be replaced by `in` and `object.keys`
package regal.rules.idiomatic["custom-has-key-construct"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result

# METADATA
# description: Missing capability for built-in function `object.keys`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_object_keys

report contains violation if {
	some rule in ast.functions

	count(rule.body) == 1

	terms := rule.body[0].terms

	terms[0].value[0].type == "var"
	terms[0].value[0].value == "eq"

	ref := _normalize_eq_terms(terms)

	ref.value[0].type == "var"

	arg_names := ast.function_arg_names(rule)

	ref.value[0].value in arg_names
	ref.value[1].type == "var"
	ref.value[1].value in arg_names

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

_normalize_eq_terms(terms) := terms[2] if {
	ast.is_wildcard(terms[1])
	terms[2].type == "ref"
}

_normalize_eq_terms(terms) := terms[1] if {
	terms[1].type == "ref"
	ast.is_wildcard(terms[2])
}
