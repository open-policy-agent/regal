# METADATA
# description: Metasyntactic variable name
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/metasyntactic-variable
package regal.rules.testing["metasyntactic-variable"]

import rego.v2

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
	some rule_index, i
	term := input.rules[rule_index].head.ref[i]

	i == 0 and term.type == "var" or term.type == "string"
	lower(term.value) in _metasyntactic

	# In case we have chained rule bodies — only flag the location where we have an actual name:
	# foo {
	#    input.x
	# } {
	#    input.y
	# }
	not ast.is_chained_rule_body(input.rules[rule_index], input.regal.file.lines)

	violation := result.fail(rego.metadata.chain(), result.location(term))
}

report contains violation if {
	some rule_index
	var := ast.found.vars[rule_index][_][_]

	lower(var.value) in _metasyntactic
	ast.is_output_var(rule_index, var)

	violation := result.fail(rego.metadata.chain(), result.location(var))
}
