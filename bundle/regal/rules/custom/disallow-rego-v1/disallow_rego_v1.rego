# METADATA
# description: Use of disallowed `import rego.v1`
package regal.rules.custom["disallow-rego-v1"]

import data.regal.ast
import data.regal.result

report contains violation if {
	ast.imports_has_path(ast.imports, ["rego", "v1"])
	violation := result.fail(rego.metadata.chain(), result.location(input.package))
}
