# METADATA
# description: Unresolved import
package regal.rules.imports["unresolved-import"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# description: collects imports and exported refs from each module
aggregate contains entry if {
	imports_with_location := [imp |
		some _import in input.imports

		_import.path.value[0].value == "data"
		len := count(_import.path.value)
		len > 1
		path := [part.value | some part in array.slice(_import.path.value, 1, len)]

		# Special case for custom rules, where we don't want to flag e.g. `import data.regal.ast`
		# as unknown, even though it's not a package included in evaluation.
		not _custom_regal_package_and_import(ast.package_path, path[0])

		imp := [path, _import.location]
	]

	exported_refs := {ast.package_path} | {ref |
		some rule in input.rules

		# locations will only contribute to each item in the set being unique,
		# which we don't want here — we only care for distinct ref paths
		some ref in _to_paths(ast.package_path, rule.head.ref)
	}

	entry := {
		"imports": imports_with_location,
		"exported_refs": exported_refs,
	}
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	all_known_refs := {path | path := input.aggregates_internal[_]["imports/unresolved-import"][_].exported_refs[_]}

	some file
	entry := input.aggregates_internal[file]["imports/unresolved-import"][_]

	some [path, location] in entry.imports
	not path in _except_imports
	not path in all_known_refs

	# cheap operation failed — need to check wildcards here to account
	# for map generating / general ref head rules
	not _wildcard_match(path, (all_known_refs | _except_imports))

	violation := result.fail(rego.metadata.chain(), {"location": object.union(util.to_location_no_text(location), {
		"file": file,
		"text": $"import data.{concat(".", path)}",
	})})
}

_custom_regal_package_and_import(pkg_path, "regal") if {
	pkg_path[0] == "custom"
	pkg_path[1] == "regal"
}

# the package part will always be included exported refs
# but if we have a rule like foo.bar.baz
# we'll want to include both foo.bar and foo.bar.baz
_to_paths(pkg_path, ref) := util.all_paths(_to_path(pkg_path, ref)) if count(ref) < 3
_to_paths(pkg_path, ref) := [_to_path(pkg_path, p) | some p in util.all_paths(ref)] if count(ref) > 2

_to_path(pkg_path, ref) := array.flatten([pkg_path, ref[0].value, [_to_string(part) |
	some part in array.slice(ref, 1, 100)
]])

_to_string(part) := part.value if part.type == "string"
_to_string(part) := "**" if part.type == "var"

_except_imports contains split(trim_prefix(str, "data."), ".") if {
	some str in config.rules.imports["unresolved-import"]["except-imports"]
}

_wildcard_match(imp_path, refs_and_except_imports) if {
	some except in refs_and_except_imports
	path := concat(".", except)
	contains(path, "*")

	# note that we are quite forgiving here, as we'll match the
	# shortest path component containing a wildcard at the end..
	# we may want to make this more strict later, but as this is
	# a new rule with a potentially high impact, let's start like
	# this and then decide if we want to be more strict later, and
	# perhaps offer that as a "strict" option
	glob.match(path, [], concat(".", imp_path))
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_aggregates[file] := agg if {
	# we know that there is only one aggregate of this type per file,
	# so we can simplify things some for our callers
	some file
	agg := input.aggregates_internal[file]["imports/unresolved-import"][_]
}
