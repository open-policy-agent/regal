# METADATA
# description: |
#   the `main` package contains the entrypoints for the linter, and routes
#   requests for linting to linter rules based on the active configuration
#   ---
#   the first step in the linting process is **prepare**, where we perform
#   validation of configuration and parameters, determine capabilities and
#   enabled linter rules, and whatever other tasks we're able to do having
#   only the information available at this stage.
#   ---
#   linter rules either **aggregate** data or **report** violations, where
#   the former is a way to find violations that can't be determined in the
#   scope of a single file
package regal.main

import data.regal.ast
import data.regal.config
import data.regal.notices
import data.regal.prepared

# METADATA
# description: |
#   set of all notices returned from linter rules
#   all notices for v1 projects run in the prepare stage, and we only run
#   the (rather expensive per-file notices if a file specifically is not v1
# scope: document
lint.notices contains notice if {
	"lint" in input.regal.operations

	prepared.file_notices

	some category, title
	_rules_to_run[category][title]

	some notice in notices.promoted_notices[category][title]
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

# METADATA
# description: prepared state for linting, after Rego preparation step
lint.prepared := prepared.prepare if "prepare" in input.regal.operations

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

	object.get(prepared.notices, [category, title], set()) == set()
	some violation in data.regal.rules[category][title].report

	not _ignored(violation, ast.ignore_directives)
}

# Check custom rules
report contains violation if {
	not _globally_ignored

	file_name_relative_to_root := trim_prefix(input.regal.file.name, concat("", [config.path_prefix, "/"]))

	some category, title
	violation := data.custom.regal.rules[category][title].report[_]

	not config.ignored_rule(category, title)
	not config.excluded_file(category, title, file_name_relative_to_root)
	not _ignored(violation, ast.ignore_directives)
}

_filename_relative_to_root := trim_prefix(input.regal.file.name, "/") if config.path_prefix == "/"
_filename_relative_to_root := trim_prefix(input.regal.file.name, concat("", [config.path_prefix, "/"])) if {
	config.path_prefix != "/"
}

_globally_ignored if {
	some compiled in data.internal.prepared.ignore_patterns.global
	glob.match(compiled, ["/"], _filename_relative_to_root)
}

_rules_to_run[category] contains title if {
	not _globally_ignored

	some category, title
	prepared.rules_to_run[category][title]

	not config.excluded_file(category, title, _filename_relative_to_root)
}

_ignored(violation, directives) if {
	ignored_rules := directives[violation.location.row]
	violation.title in ignored_rules
}

_ignored(violation, directives) if {
	ignored_rules := directives[violation.location.row + 1]
	violation.title in ignored_rules
}
