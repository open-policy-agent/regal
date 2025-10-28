# METADATA
# description: Importing own package is pointless
package regal.rules.imports["pointless-import"]

import data.regal.ast
import data.regal.result

# METADATA
# description: report pointless imports of own package or rules defined in the same module
# scope: document

# METADATA
# description: report pointless imports of own package
report contains violation if {
	path := input.imports[_].path

	ast.ref_value_equal(input.package.path, path.value)

	violation := result.fail(rego.metadata.chain(), result.location(path))
}

# METADATA
# description: report pointless imports of rule paths defined in the same module
report contains violation if {
	rule_paths := {path |
		rref := input.rules[_].head.ref
		path := ast.extend_ref_terms(input.package.path, rref)
	}
	imp_path := input.imports[_].path

	some rule_path in rule_paths
	ast.is_terms_subset(imp_path.value, rule_path)

	violation := result.fail(rego.metadata.chain(), result.location(imp_path))
}
