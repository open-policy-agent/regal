# METADATA
# description: Missing entrypoint annotation
package regal.rules.idiomatic["no-defined-entrypoint"]

import data.regal.result
import data.regal.util

# METADATA
# description: |
#   collects `entrypoint: true` annotations from any given module
# scope: document
aggregate contains entry if {
	some i
	input.package.annotations[i].entrypoint == true

	entry := {"entrypoint": util.to_location_object(input.package.annotations[i].location)}
}

aggregate contains entry if {
	some i, j
	input.rules[i].annotations[j].entrypoint == true

	entry := {"entrypoint": util.to_location_object(input.rules[i].annotations[j].location)}
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	not _any_entrypoint

	violation := result.fail(rego.metadata.chain(), {})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_any_entrypoint if input.aggregates_internal[_]["idiomatic/no-defined-entrypoint"][_].entrypoint
