# METADATA
# description: Prefer using `object.union_n` over nested `object.union` calls
package regal.rules.idiomatic["use-object-union-n"]

import data.regal.ast
import data.regal.config
import data.regal.result

# METADATA
# description: Missing capability for built-in `object.union` and/or `object.union_n`
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	not object.subset(object.keys(config.capabilities.builtins), {"object.union", "object.union_n"})
}

report contains violation if {
	some call in nested_union_calls

	violation := result.fail(rego.metadata.chain(), result.location(call))
}

report contains violation if {
	config.rules.idiomatic["use-object-union-n"]["flag-all-union"] == true

	some call in all_object_union_calls

	violation := json.patch(result.fail(rego.metadata.chain(), result.location(call)), [{
		"op": "replace",
		"path": "/description",
		"value": "Prefer using `object.union_n` over `object.union`",
	}])
}

# METADATA
# description: |
#   instead of:
#     - `object.union(a, object.union(b, c))`
#   recommend:
#     - `object.union_n([a, b, c])`
nested_union_calls contains call if {
	some call in all_object_union_calls
	some pos in [1, 2]

	call[pos].type == "call"

	arg := call[pos].value[0]

	arg.value[0].value == "object"
	arg.value[1].value == "union"
}

# METADATA
# description: |
#   optionally (when 'flag-all-union' is enabled in configuration),
#   recommend replacing all calls to `object.union` with `object.union_n`
all_object_union_calls contains call if {
	"object.union" in ast.builtin_functions_called

	some calls in ast.found.calls
	some call in calls

	call[0].value[0].value == "object"
	call[0].value[1].value == "union"
}
