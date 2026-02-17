# METADATA
# description: Package or rule missing metadata
package regal.rules.custom["missing-metadata"]

import data.regal.ast
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# description: aggregates annotations on package declarations and rules
aggregate contains {
	"package_annotated": _package_annotated,
	"package_location": util.to_location_object(input.package.location),
	"rule_annotations": _rule_annotations,
	"rule_locations": _rule_locations,
	"package_name": ast.package_name,
	"file": input.regal.file.name,
}

default _package_annotated := false

_package_annotated if input.package.annotations

_rule_annotations[rule_path] contains annotated if {
	some rule in ast.public_rules_and_functions

	rule_path := concat(".", [ast.package_name, ast.ref_static_to_string(rule.head.ref)])
	annotated := object.get(rule, "annotations", []) != []
}

_rule_locations[rule_path] := location if {
	some rule_path, annotated in _rule_annotations

	# we only care about locations of non-annotated rules
	not true in annotated

	first_rule_index := [i |
		some i

		# false positive: https://github.com/open-policy-agent/regal/issues/1353
		# regal ignore:unused-output-variable
		ref := ast.public_rules_and_functions[i].head.ref
		concat(".", [ast.package_name, ast.ref_static_to_string(ref)]) == rule_path
	][0]

	ref := ast.public_rules_and_functions[first_rule_index].head.ref
	location := object.remove(result.ranged_from_ref(ref).location, ["file"])
}

# METADATA
# description: report packages without metadata
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	cfg := config.rules.custom["missing-metadata"]

	some pkg_path, aggs in _package_path_aggs
	every item in aggs {
		item[1].package_annotated == false
	}

	not _excepted_package_pattern(cfg, pkg_path)

	first_item := [item[1] | some item in aggs][0]

	violation := result.fail(rego.metadata.chain(), {"location": object.union(first_item.package_location, {
		"file": first_item.file,
		"text": first_item.package_location.text,
	})})
}

# METADATA
# description: report rules without metadata annotationsfile
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	cfg := config.rules.custom["missing-metadata"]

	some rule_path, aggregates in _rule_path_aggs

	every aggregate in aggregates {
		aggregate.annotated == false
	}

	not _excepted_rule_pattern(cfg, rule_path)

	any_item := util.any_set_item(aggregates)

	violation := result.fail(rego.metadata.chain(), {"location": object.union(any_item.location, {
		"file": any_item.file,
		"text": split(any_item.location.text, "\n")[0],
	})})
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_package_path_aggs[item.package_name] contains [file, item] if {
	some file
	item := input.aggregates_internal[file]["custom/missing-metadata"][_]
}

# METADATA
# schemas:
#   - input: schema.regal.aggregate
_rule_path_aggs[rule_path] contains agg if {
	some file
	item := input.aggregates_internal[file]["custom/missing-metadata"][_]
	some rule_path, annotations in item.rule_annotations

	agg := {
		"file": file,
		"location": item.rule_locations[rule_path],
		"annotated": true in annotations,
	}
}

_excepted_package_pattern(cfg, value) if regex.match(cfg["except-package-path-pattern"], value)

_excepted_rule_pattern(cfg, value) if regex.match(cfg["except-rule-path-pattern"], value)
