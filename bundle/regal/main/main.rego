# METADATA
# description: |
#   the `main` package contains the entrypoints for linting, and routes
#   requests for linting to linter rules based on the active configuration
#   ---
#   linter rules either **aggregate** data or **report** violations, where
#   the former is a way to find violations that can't be determined in the
#   scope of a single file
package regal.main

import data.regal.ast
import data.regal.config
import data.regal.notices
import data.regal.util

# METADATA
# description: set of all notices returned from linter rules
lint.notices contains notice if {
	"lint" in input.regal.operations

	some category, title
	_rules_to_run[category][title]

	rule_notices := notices.promoted_notices[category][title]
	some notice in rule_notices
}

# METADATA
# description: map of all ignore directives encountered when linting
lint.ignore_directives[input.regal.file.name] := ast.ignore_directives if "lint" in input.regal.operations

# METADATA
# description: all violations from non-aggregate rules
lint.violations := report if "lint" in input.regal.operations

# METADATA
# description: map of all aggregated data from aggregate rules, keyed by category/title
lint.aggregates := aggregate if "collect" in input.regal.operations

# METADATA
# description: all violations from aggregate rules
lint.aggregate.violations := aggregate_report if "aggregate" in input.regal.operations

_file_name_relative_to_root(filename, "/") := trim_prefix(filename, "/")
_file_name_relative_to_root(filename, root) := trim_prefix(filename, concat("", [root, "/"])) if root != "/"

# METADATA
# description: |
#   set of all rules not disabled by configuration
#   note that this only accounts for rules disabled entirely, not for specific files, or via flags
enabled_rules[category][title] if {
	some category, title
	config.rules[category][title]

	not config.ignored_rule(category, title)
}

_rules_to_run[category] contains title if {
	relative_filename := _file_name_relative_to_root(input.regal.file.name, config.path_prefix)
	not config.ignored_globally(relative_filename)

	some category, title
	config.rules[category][title]

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, relative_filename)
}

# METADATA
# title: report
# description: |
#   This is the main entrypoint for linting, The report rule runs all rules against an input AST and produces a report
# entrypoint: true
report contains violation if {
	not is_object(input)

	violation := {
		"category": "error",
		"title": "invalid-input",
		"description": "provided input must be a JSON AST",
	}
}

report contains violation if {
	not input.package

	violation := {
		"category": "error",
		"title": "invalid-input",
		"description": "provided input must be a JSON AST",
	}
}

# Check bundled rules
report contains violation if {
	some category, title
	_rules_to_run[category][title]

	count(object.get(notices.promoted_notices, [category, title], [])) == 0

	some violation in data.regal.rules[category][title].report

	not _ignored(violation, ast.ignore_directives)
}

# Check custom rules
report contains violation if {
	file_name_relative_to_root := trim_prefix(input.regal.file.name, concat("", [config.path_prefix, "/"]))
	not config.ignored_globally(file_name_relative_to_root)

	some category, title

	violation := data.custom.regal.rules[category][title].report[_]

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, file_name_relative_to_root)
	not _ignored(violation, ast.ignore_directives)
}

# METADATA
# description: collects aggregates in bundled rules
# scope: rule
aggregate[category_title] contains entry if {
	some category, title
	_rules_to_run[category][title]

	some entry in data.regal.rules[category][title].aggregate

	category_title := concat("/", [category, title])
}

# METADATA
# description: collects aggregates in custom rules
# scope: rule
aggregate[category_title] contains entry if {
	not config.ignored_globally(input.regal.file.name)

	some category, title

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, input.regal.file.name)

	entries := _mark_if_empty(data.custom.regal.rules[category][title].aggregate)

	category_title := concat("/", [category, title])

	some entry in entries
}

# a custom aggregate rule may not come back with entries, but we still need
# to register the fact that it was called so that we know to call the
# aggregate_report for the same rule later
#
# for these cases we just return an empty map, and let the aggregator on the Go
# side handle this case
_mark_if_empty(entries) := {{}} if {
	count(entries) == 0
} else := entries

# METADATA
# description: Check bundled rules using aggregated data
# schemas:
#   - input: schema.regal.aggregate
aggregate_report contains violation if {
	some category, title
	_rules_to_run[category][title]

	key := concat("/", [category, title])
	input_for_rule := object.remove(
		object.union(input, {"aggregate": object.get(input, ["aggregates_internal", key], [])}),
		["aggregates_internal"],
	)

	# regal ignore:with-outside-test-context
	some violation in data.regal.rules[category][title].aggregate_report with input as input_for_rule

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
	not config.ignored_globally(input.regal.file.name)

	some key in object.keys(input.aggregates_internal)
	[category, title] := split(key, "/")

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, input.regal.file.name)

	input_for_rule := object.remove(
		object.union(input, {"aggregate": _null_to_empty(input.aggregates_internal[key])}),
		["aggregates_internal"],
	)

	# regal ignore:with-outside-test-context
	some violation in data.custom.regal.rules[category][title].aggregate_report with input as input_for_rule

	# don't assume that the author included a location in the violation, although they really should
	ignore_directives := object.get(input, ["ignore_directives", object.get(violation, ["location", "file"], "")], {})

	not _ignored(violation, util.keys_to_numbers(ignore_directives))
}

_ignored(violation, directives) if {
	ignored_rules := directives[util.to_location_object(violation.location).row]
	violation.title in ignored_rules
}

_ignored(violation, directives) if {
	ignored_rules := directives[util.to_location_object(violation.location).row + 1]
	violation.title in ignored_rules
}

_null_to_empty(x) := [] if {
	x == null
} else := x
