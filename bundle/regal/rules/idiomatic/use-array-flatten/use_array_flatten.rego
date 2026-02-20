# METADATA
# description: Prefer using `array.flatten` over nested `array.concat` calls
package regal.rules.idiomatic["use-array-flatten"]

import data.regal.ast
import data.regal.config
import data.regal.result

# METADATA
# description: Missing capability for built-in `array.flatten` and/or `array.concat`
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	not object.subset(object.keys(config.capabilities.builtins), {"array.flatten", "array.concat"})
}

report contains violation if {
	not config.rules.idiomatic["use-array-flatten"]["flag-all-concat"]

	some call in nested_concat_calls

	violation := result.fail(rego.metadata.chain(), result.location(call))
}

report contains violation if {
	not config.rules.idiomatic["use-array-flatten"]["flag-all-concat"]
	config.rules.idiomatic["use-array-flatten"]["flag-wrapped-concat"] == true

	some call in wrapped_concat_calls

	violation := json.patch(result.fail(rego.metadata.chain(), result.location(call)), [{
		"op": "replace",
		"path": "/description",
		"value": "Prefer `array.flatten` over `array.concat` with array literals wrapping arguments",
	}])
}

report contains violation if {
	config.rules.idiomatic["use-array-flatten"]["flag-all-concat"] == true

	some call in all_concat_calls

	violation := json.patch(result.fail(rego.metadata.chain(), result.location(call)), [{
		"op": "replace",
		"path": "/description",
		"value": "Prefer using `array.flatten` over `array.concat`",
	}])
}

# METADATA
# description: |
#   instead of:
#     - `array.concat(a, array.concat(b, c))`
#   recommend:
#     - `array.flatten([a, b, c])`
nested_concat_calls contains call if {
	some call in all_concat_calls
	some pos in [1, 2]

	call[pos].type == "call"

	arg := call[pos].value[0]

	arg.value[0].value == "array"
	arg.value[1].value == "concat"
}

# METADATA
# description: |
#  optionally (when 'flag-wrapped-concat' is enabled in configuration), instead of:
#  - `array.concat([a], b)`
#  recommend:
#  - `array.flatten([a, b])`
wrapped_concat_calls contains call if {
	some call in all_concat_calls
	some pos in [1, 2]

	call[pos].type == "array"
}

# METADATA
# description: |
#  optionally (when 'flag-all-concat' is enabled in configuration),
#  recommend replacing all calls to `array.concat` with `array.flatten`
all_concat_calls contains call if {
	"array.concat" in ast.builtin_functions_called

	some calls in ast.found.calls
	some call in calls

	call[0].value[0].value == "array"
	call[0].value[1].value == "concat"
}
