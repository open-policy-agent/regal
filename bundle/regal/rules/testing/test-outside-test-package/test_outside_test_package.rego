# METADATA
# description: Test outside of test package
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/testing/test-outside-test-package
package regal.rules.testing["test-outside-test-package"]

import data.regal.ast
import data.regal.result

report contains violation if {
	not _is_test_package(ast.package_name)

	some rule in ast.tests

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

_is_test_package(package_name) if endswith(package_name, "_test")
_is_test_package("test")
