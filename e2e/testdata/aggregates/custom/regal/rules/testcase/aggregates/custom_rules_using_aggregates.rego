# METADATA
# description: Collect data in aggregates and validate it
package custom.regal.rules.testcase.aggregates

import data.regal.result

# METADATA
# description: Collect aggregates
aggregate contains {}

# METADATA
# description: Validate that two files were processed
aggregate_report contains violation if {
	not _two_files_processed

	violation := result.fail(rego.metadata.chain(), {})
}

_two_files_processed if count(input.aggregate) == 2
