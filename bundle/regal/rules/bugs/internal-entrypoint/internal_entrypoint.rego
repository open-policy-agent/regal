# METADATA
# description: Entrypoint can't be marked internal
package regal.rules.bugs["internal-entrypoint"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some rule in ast.rules
	some annotation in rule.annotations

	annotation.entrypoint == true

	some i, term in rule.head.ref

	startswith(term.value, "_")
	true in {
		i == 0,
		term.type == "string",
	}

	violation := result.fail(rego.metadata.chain(), result.location(term))
}
