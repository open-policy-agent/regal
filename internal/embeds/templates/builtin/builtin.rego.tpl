# METADATA
# description: Add description of rule here!
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/{{.Category}}/{{.NameOriginal}}
package regal.rules.{{.Category}}{{.Name}}

import data.regal.ast
import data.regal.result

report contains violation if {
	# Or change to imports, packages, comments, etc.
	some rule in input.rules

	# Deny any rule named foo, bar, or baz. This is just an example!
	# Add your own rule logic here.
	ast.ref_to_string(rule.head.ref) in {"foo", "bar", "baz"}

	violation := result.fail(rego.metadata.chain(), result.location(rule))
}
