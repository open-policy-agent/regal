# METADATA
# description: Prefer importing packages over rules
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/imports/prefer-package-imports
package regal.rules.imports["prefer-package-imports"]

import future.keywords.not

import data.regal.aggregated
import data.regal.config
import data.regal.result

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	some file
	entry := input.aggregates_internal[file].common[_]

	some [path, location] in entry.imports

	not path in aggregated.all_package_paths
	not path in _ignored_import_paths

	[path |
		some pkg_path in aggregated.all_package_paths
		pkg_path == array.slice(path, 0, count(pkg_path))
	] != []

	violation := result.fail(rego.metadata.chain(), aggregated.location_object(location, file))
}

_ignored_import_paths contains split(trim_prefix(item, "data."), ".") if {
	some item in config.rules.imports["prefer-package-imports"]["ignore-import-paths"]
}
