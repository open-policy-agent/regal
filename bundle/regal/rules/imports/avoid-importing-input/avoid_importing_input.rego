# METADATA
# description: Avoid importing input
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/avoid-importing-input
package regal.rules.imports["avoid-importing-input"]

import future.keywords.not

import data.regal.result

report contains violation if {
	some i
	input.imports[i].path.value[0].value == "input"

	# Allow aliasing input, eg `import input as tfplan`:
	not {
		count(input.imports[i].path.value) == 1
		input.imports[i].alias
	}

	violation := result.fail(rego.metadata.chain(), result.location(input.imports[i].path.value[0]))
}

_aliased_input(imported) if {
	count(imported.path.value) == 1
	imported.alias
}
