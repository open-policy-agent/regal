# METADATA
# description: Max file length exceeded
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/file-length
package regal.rules.style["file-length"]

import data.regal.config
import data.regal.result

report contains violation if {
	count(input.regal.file.lines) > config.rules.style["file-length"]["max-file-length"]

	violation := result.fail(rego.metadata.chain(), result.location(input.package))
}
