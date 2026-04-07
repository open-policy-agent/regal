# METADATA
# description: Rule named "if"
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/bugs/rule-named-if
package regal.rules.bugs["rule-named-if"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result

# METADATA
# description: Since OPA 1.0, rule-named-if enabled only when provided a v0 policy
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

report contains violation if {
	# this is only an optimization — as we already have collected all the rule
	# names, we'll do a fast lookup to know if we need to iterate over the rules
	# at all, which we'll do only to retrieve the location of the rule
	"if" in ast.rule_names

	some rule in input.rules
	ast.ref_to_string(rule.head.ref) == "if"

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}
