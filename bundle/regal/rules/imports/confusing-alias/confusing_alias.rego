# METADATA
# description: Confusing alias of existing import
package regal.rules.imports["confusing-alias"]

import data.regal.ast
import data.regal.result

report contains violation if {
	some aliased in _aliased_imports
	some imp in input.imports

	imp != aliased
	count(aliased.path.value) == count(imp.path.value)
	ast.is_terms_subset(aliased.path.value, imp.path.value)

	violation := result.fail(rego.metadata.chain(), result.location(aliased))
}

_aliased_imports contains imp if {
	some imp in input.imports

	imp.alias
}
