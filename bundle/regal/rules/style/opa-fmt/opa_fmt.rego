# METADATA
# description: File should be formatted with `opa fmt`
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/opa-fmt
package regal.rules.style["opa-fmt"]

import data.regal.result

report contains violation if {
	# NOTE:
	# 1. this won't identify CRLF line endings, as we've stripped them from the input previously
	# 2. this will perform worse than having the text representation of the file in the input
	not regal.is_formatted(concat("\n", input.regal.file.lines), {"rego_version": input.regal.file.rego_version})

	violation := result.fail(rego.metadata.chain(), {"location": {
		"file": input.regal.file.name,
		"row": 1,
		"col": 1,
		"text": input.regal.file.lines[0],
	}})
}
