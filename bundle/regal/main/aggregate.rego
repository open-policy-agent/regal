package regal.main

import data.regal.aggregators
import data.regal.ast
import data.regal.config
import data.regal.prepared
import data.regal.util

# METADATA
# description: collect common data used by aggregate rules
aggregate[input.regal.file.name].common contains {
	"package_name": ast.package_name,
	"package_name_full": ast.package_name_full,
	"package_path": ast.package_path,
	"lines": input.regal.file.lines,
	"rule_tree": aggregators.rule_tree,
	"imports": aggregators.imports,
}

# METADATA
# description: collects aggregates in bundled rules
# scope: rule
aggregate[input.regal.file.name][key] contains entry if {
	some category, title
	_rules_to_run[category][title]

	key := prepared.aggregate_keys[category][title]

	some entry in _mark_if_empty(data.regal.rules[category][title].aggregate)
}

# METADATA
# description: collects aggregates in custom rules
# scope: rule
aggregate[input.regal.file.name][$"{category}/{title}"] contains entry if {
	not _globally_ignored

	some category, title

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, input.regal.file.name)

	some entry in _mark_if_empty(data.custom.regal.rules[category][title].aggregate)
}

# METADATA
# description: Check bundled rules using aggregated data
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	some category, title
	_rules_to_run[category][title]

	some violation in data.regal.rules[category][title].aggregate_report

	not _ignored(violation, util.keys_to_numbers(object.get(
		input.ignore_directives,
		# some aggregate violations won't have a location at all, like no-defined-entrypoint
		object.get(violation, ["location", "file"], ""),
		{},
	)))
}

# METADATA
# description: Check custom rules using aggregated data
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	data.custom.regal

	some key, aggregate in _aggregate_report_inputs
	[category, title] := split(key, "/")

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, input.regal.file.name)

	input_for_rule := _remove_empty_aggregates(aggregate)

	# regal ignore:with-outside-test-context
	some violation in data.custom.regal.rules[category][title].aggregate_report with input as input_for_rule

	# don't assume that the author included a location in the violation, although they really should
	ignore_directives := object.get(input, ["ignore_directives", object.get(violation, ["location", "file"], "")], {})

	not _ignored(violation, util.keys_to_numbers(ignore_directives))
}

_remove_empty_aggregates(aggregates) := {"aggregate": set()} if {
	aggregates == {"aggregate": {set()}}
} else := aggregates

# METADATA
# description: |
#   Restructure aggregate report input for conformance with "legacy"
#   format, so that we don't break existing rules. Later on we can deprecate
#   this and make it opt-in.
# schemas:
#   - input: schema.regal.aggregate
_aggregate_report_inputs[cat_title].aggregate contains formatted if {
	some filename, cat_title
	aggregate := input.aggregates_internal[filename][cat_title][_]
	formatted := _format_aggregate(aggregate, filename)
}

# a custom aggregate rule may not come back with entries, but we still need
# to register the fact that it was called so that we know to call the
# aggregate_report for the same rule later
#
# for these cases we just return an empty map, and let the aggregator on the Go
# side handle this case
_mark_if_empty(entries) := {{}} if {
	entries == set()
} else := entries

default _format_aggregate(_, _) := set()

_format_aggregate(aggregate, filename) := object.union(aggregate, {"aggregate_source": {"file": filename}}) if {
	aggregate.aggregate_data
}
