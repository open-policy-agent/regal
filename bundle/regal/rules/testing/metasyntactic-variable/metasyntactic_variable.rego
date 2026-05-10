# METADATA
# description: Metasyntactic variable name
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/metasyntactic-variable
package regal.rules.testing["metasyntactic-variable"]

import data.regal.ast
import data.regal.result

_metasyntactic := {
	"foobar",
	"foo",
	"bar",
	"baz",
	"qux",
	"quux",
	"corge",
	"grault",
	"garply",
	"waldo",
	"fred",
	"plugh",
	"xyzzy",
	"thud",
}

report contains violation if {
	some rule in input.rules
	some term in ast.named_refs(rule.head.ref)

	lower(term.value) in _metasyntactic

	# In case we have chained rule bodies — only flag the location where we have an actual name:
	# foo {
	#    input.x
	# } {
	#    input.y
	# }
	not ast.is_chained_rule_body(rule, input.regal.file.lines)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}

report contains violation if {
	some i
	var := ast.found.vars[i][_][_]

	lower(var.value) in _metasyntactic

	ast.is_output_var(input.rules[i], var)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}
