# METADATA
# description: Outside reference to internal rule or function
package regal.rules.bugs["leaked-internal-reference"]

import data.regal.ast
import data.regal.config
import data.regal.result

report contains violation if {
	_enabled_for_file

	value := ast.found.refs[_][_].value

	some term in value

	term.type == "string"
	startswith(term.value, "_")

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(value))
}

report contains violation if {
	_enabled_for_file

	value := input.imports[_].path.value

	some term in value

	term.type == "string"
	startswith(term.value, "_")

	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(value))
}

_enabled_for_file if not _test_file

_enabled_for_file if {
	_test_file
	config.rules.bugs["leaked-internal-reference"]["include-test-files"]
}

_test_file if input.regal.file.name == "test.rego"
_test_file if endswith(input.regal.file.name, "_test.rego")
