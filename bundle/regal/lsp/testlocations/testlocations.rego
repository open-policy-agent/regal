# METADATA
# description: |
#   This returns a set of test_ rule locations in a given module.
#   Used by the regal/testLocations method in the LSP to have clients know
#   where tests are.
# schemas:
#   - input: schema.regal.ast
package regal.lsp.testlocations

import data.regal.ast
import data.regal.result as rs

# METADATA
# description: |
#   result contains a list of locations. A location is test name, package and
#   the location (which has start and end char range too).
result contains object.union(loc, {
	"package_path": ast.package_path,
	"package": _package_ref_string,
	"name": ast.ref_static_to_string(rule.head.ref),
	"root": input.regal.file.root,
}) if {
	some rule in ast.tests

	loc := rs.location(rule.head)
}

_package_ref_string := ast.ref_to_string(input.package.path)
