# METADATA
# description: Add description of aggregate rule here!
package custom.regal.rules.{{.Category}}{{.Name}}

import data.regal.ast
import data.regal.result

# METADATA
# description: collects per-module data for the aggregate_report rule
aggregate contains entry if {
	# Collect data about this module that the aggregate_report rule will
	# correlate across all modules. Add your own collection logic here.
	some rule in input.rules

	entry := result.aggregate(rego.metadata.chain(), {
		"ref": ast.ref_to_string(rule.head.ref),
		"location": result.location(rule),
	})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	# Correlate the collected entries across modules and emit a violation.
	# Replace this example with your own cross-module condition.
	some entry in input.aggregate

	violation := result.fail(rego.metadata.chain(), entry.aggregate_source)
}
