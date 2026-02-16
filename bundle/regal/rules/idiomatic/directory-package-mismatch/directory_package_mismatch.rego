# METADATA
# description: Directory structure should mirror package
package regal.rules.idiomatic["directory-package-mismatch"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# description: disabled when filename is unknown
# custom:
#   severity: warn
notices contains result.notice(rego.metadata.chain()) if "no_filename" in config.capabilities.special

report contains violation if {
	parts := split(input.regal.file.abs, input.regal.environment.path_separator)
	file_path_values := array.slice(parts, 0, count(parts) - 1)

	# get the last n components from file path, where n == count(_pkg_path_values)
	file_path_length_matched := array.slice(
		file_path_values,
		count(file_path_values) - count(_pkg_path_values),
		count(file_path_values),
	)

	file_path_length_matched != _pkg_path_values

	# skip the "data" part of the path, as it has no location
	violation := result.fail(rego.metadata.chain(), result.ranged_from_ref(util.rest(input.package.path)))
}

_pkg_path_values := without_test_suffix if {
	config.rules.idiomatic["directory-package-mismatch"]["exclude-test-suffix"] == true

	name := regal.last(ast.package_path)

	endswith(name, "_test")

	without_test_suffix := array.flatten([
		array.slice(ast.package_path, 0, count(ast.package_path) - 1),
		trim_suffix(name, "_test"),
	])
} else := ast.package_path
