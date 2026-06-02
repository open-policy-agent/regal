package regal.aggregators

import future.keywords.not

import data.regal.ast
import data.regal.util

# METADATA
# description: aggregates a rule tree for the given input module
rule_tree := tree if {
	pkg_name := replace(replace(ast.package_name_full, `["`, "."), `"]`, "")
	pkg_path := split(pkg_name, ".")

	pkg_obj := json.patch({}, [patch |
		some i in numbers.range(1, count(pkg_path))

		patch := {
			"op": "add",
			"path": $"/{concat("/", array.slice(pkg_path, 0, i))}",
			"value": {},
		}
	])

	tree := json.patch(pkg_obj, [patch |
		some rule_path in _rule_paths

		patch := {
			"op": "add",
			"path": $"/{concat("/", array.concat(pkg_path, rule_path))}",
			"value": {},
		}
	])
}

# METADATA
# description: |
#   aggregates all input refs and their locations for the given input module§
imports := [imp |
	some _import in input.imports

	_import.path.value[0].value == "data"
	len := count(_import.path.value)
	len > 1

	# Special case for custom rules, where we don't want to flag e.g. `import data.regal.ast`
	# as unknown, even though it's not a package included in evaluation.
	not {
		_import.path.value[1].value == "regal"
		ast.package_path[0] == "custom"
		ast.package_path[1] == "regal"
	}

	path := [term.value | some term in array.slice(_import.path.value, 1, len)]
	imp := [path, _import.location]
]

_rule_paths contains path if {
	pkg_name := replace(replace(ast.package_name_full, `["`, "."), `"]`, "")
	pkg_pref := $"{pkg_name}."

	some name, _ in ast.rule_head_locations
	some path in util.all_paths(split(trim_prefix(name, pkg_pref), "."))
}
