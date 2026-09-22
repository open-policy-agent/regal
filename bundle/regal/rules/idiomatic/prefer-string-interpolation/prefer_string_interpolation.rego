# METADATA
# description: Prefer string interpolation
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/idiomatic/prefer-string-interpolation
package regal.rules.idiomatic["prefer-string-interpolation"]

import future.keywords.not
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
# description: Prefer string interpolation over `sprintf`
# scope: rule
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
	_include_non_var_args or {
		every arg in fun.args[1].value {
			arg.type == "var"
		}
	}

	violation := result.fail(rego.metadata.chain(), result.location(fun.location))
}

# METADATA
# description: Prefer string interpolation over `concat`
# scope: rule
report contains violation if {
	some rule_index, fun
	ast.function_calls[rule_index][fun].name == "concat"

	fun.args[1].type == "array" # only array literals, not vars or refs
	_include_non_var_args or {
		every arg in fun.args[1].value {
			arg.type == "var" or arg.type == "string"
		}
	}

	not {
		# concat sometimes used to break up long static strings in places
		# where raw strings don't work well, like text containing multiple
		# backticks - don't suggest interpolation in this case
		delim := fun.args[0].value
		delim == "\n" or delim == ""

		every arg in fun.args[1].value {
			arg.type == "string"
		}
	}

	# to consider:
	# perhaps we should have an option for max number of arguments to consider for interpolation?
	# or better (but complex) — to measure the length of the resulting string and decide based on that?
	# it's not ideal if we recommend interpolation in cases where the string would be extremely long

	violation := result.fail(rego.metadata.chain(), result.location(fun.location))
}

_include_non_var_args if config.rules.idiomatic["prefer-string-interpolation"]["include-non-var-args"] == true
