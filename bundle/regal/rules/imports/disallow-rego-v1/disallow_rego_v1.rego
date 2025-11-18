# METADATA
# description: Use `import rego.v1`
package regal.rules.imports["disallow-rego-v1"]

import data.regal.ast
import data.regal.capabilities
import data.regal.result

# METADATA
# description: Missing capability for `import rego.v1`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if {
	not capabilities.has_rego_v1_feature
	not capabilities.is_opa_v1
}

# METADATA
# description: Since OPA 1.0, use-rego-v1 enabled only when provided a v0 policy
# custom:
#   severity: none
notices contains result.notice(rego.metadata.chain()) if {
	capabilities.is_opa_v1
	input.regal.file.rego_version != "v0"
}

report contains violation if {
	capabilities.is_opa_v1
	ast.imports_has_path(ast.imports, ["rego", "v1"])
	violation := result.fail(rego.metadata.chain(), result.location(input.package))
}
