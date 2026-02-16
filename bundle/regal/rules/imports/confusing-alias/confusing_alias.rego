# METADATA
# description: Confusing alias of existing import
package regal.rules.imports["confusing-alias"]

import data.regal.result

report contains violation if {
	some aliased in _aliased_imports
	some imp in input.imports

	imp != aliased
	_paths_equal(aliased.path.value, imp.path.value)

	violation := result.fail(rego.metadata.chain(), result.location(aliased))
}

_aliased_imports contains imp if {
	some imp in input.imports

	imp.alias
}

_paths_equal(p1, p2) if {
	count(p1) == count(p2)

	every i, part in p1 {
		part.type == p2[i].type
		part.value == p2[i].value
	}
}
