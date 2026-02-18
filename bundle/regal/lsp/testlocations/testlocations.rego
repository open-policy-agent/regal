package regal.lsp.testlocations

import data.regal.ast
import data.regal.result

result contains test if {
	some test in _single_tests
}

result contains object.union(loc, {
	"package": _package_ref_string,
	"kind": kind,
}) if {
	count(_single_tests) > 0

	# package_test is run all tests in package, file_test, is run all in file
	some kind in {"package_test", "file_test"}

	some test in ast.tests

	loc := result.location(input.package)
}

_single_tests contains object.union(loc, {
	"package": _package_ref_string,
	"kind": "single_test",
	"test": ast.ref_static_to_string(test.head.ref),
}) if {
	some test in ast.tests

	loc := result.location(test.head)
}

_package_ref_string := ast.ref_to_string(input.package.path)
