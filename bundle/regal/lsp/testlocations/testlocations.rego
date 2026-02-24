# METADATA
# description: |
#   This returns a set of test_ rule locations in a given module.
#   Used by the regal/testLocations method in the LSP to have clients know
#   where tests are.
package regal.lsp.testlocations

import data.regal.ast
import data.regal.result as rs

# METADATA
# description: |
#   result contains a list of locations. A location is test name, package and
#   the location (which has start and end char range too).
result contains object.union(loc, {
	"package": _package_ref_string,
	"name": ast.ref_static_to_string(test.head.ref),
}) if {
	some test in ast.tests

	loc := rs.location(test.head)
}

_package_ref_string := ast.ref_to_string(input.package.path)
