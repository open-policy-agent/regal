# METADATA
# description: Prefer string interpolation where possible
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation
package regal.rules.idiomatic["prefer-string-interpolation"]

import future.keywords.or

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.result

# METADATA
# description: Capabilities missing template string feature
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_string_interpolation

# METADATA
# description: Capabilities missing sprintf built-in function
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_sprintf

report contains violation if {
	some rule_index, fun
	ast.function_calls[rule_index][fun].name == "sprintf"

	fun.args[0].type == "string"
	fun.args[1].type == "array"

	str_no_escape := replace(fun.args[0].value, "%%", "")

	# for now, only if all patterns match either %s, &q, %d and %v (optionally,
	# with # in front of it) do we recommend using interpolation instead
	regex.match(`%#?[sqdv]{1}`, str_no_escape)
	not regex.match(`%#?[^sqdv]`, str_no_escape)

	# simple variable reference args (e.g. `[x]`) are guaranteed to be defined by compiler,
	# but whether to also flag calls like `sprintf("%s", [input.x])` (argument contains ref)
	# must be up to the user to opt in to, as replacing such calls with interpolation changes
	# the semantics of policy evaluation for potentially undefined references:
	# - `sprintf` = evaluation fails on undefined ref
	# - interpolation = writes `<undefined>` in place of ref
	{
		config.rules.idiomatic["prefer-string-interpolation"]["include-non-var-args"] == true
	} or {
		every arg in fun.args[1].value {
			arg.type == "var"
		}
	}

	violation := result.fail(rego.metadata.chain(), result.location(fun.location))
}
