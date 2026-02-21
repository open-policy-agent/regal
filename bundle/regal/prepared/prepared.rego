# METADATA
# description: |
#   package containing logic run during the prepare stage, which as the name implies
#   runs before any linters. most of the data available at lint time is also available
#   here, with the notable exception of input files to lint
package regal.prepared

import data.regal.config
import data.regal.notices as ext_notices

# METADATA
# description: rules determined to run after accounting for configuration settings and overrides
prepare.rules_to_run[category] contains title if {
	some category, title
	config.rules[category][title]

	not config.ignored_rule(category, title)
}

# METADATA
# description: |
#   notices collected during the prepare stage, which avoids having to re-run the same rules
#   for each file even though the results will be identical. at this stage we assume rego v1
#   as that is likely the common case for Regal-enabled projects now. only when we encounter
#   non-v1 files during linting will we re-run the notice-collection, and only for files of
#   that version.
prepare.notices[category][title] contains notice if {
	some category, title
	prepare.rules_to_run[category][title]

	some notice in ext_notices.promoted_notices[category][title]
}

# METADATA
# description: store keys for aggregate rules to avoid repeating it for each input file
prepare.aggregate_keys[category][title] := concat("/", [category, title]) if {
	some category, title
	prepare.rules_to_run[category][title]
}

# METADATA
# description: compile ignore patterns found for rules in config
prepare.ignore_patterns.files[category][title] contains compiled if {
	some category, titles in prepare.rules_to_run
	some title in titles
	some compiled in config.patterns_compiler(config.rules[category][title].ignore.files)
}

# METADATA
# description: compile global ignore patterns found in config and params
prepare.ignore_patterns.global := config.patterns_compiler(data.eval.params.ignore_files) if {
	data.eval.params.ignore_files
} else := config.patterns_compiler(config.merged_config.ignore.files)

# METADATA
# description: once prepared, rules_to_run fetched from storage
# scope: document
default rules_to_run := {}

rules_to_run := data.internal.prepared.rules_to_run

# METADATA
# description: once prepared, notices fetched from storage
# scope: document
default notices := {}

notices := data.internal.prepared.notices if not file_notices

notices := ext_notices.promoted_notices if file_notices

# METADATA
# description: determine if per-file notices should be processed or fetched from prepared state
# scope: document
file_notices if input.regal.file.rego_version != "v1"
file_notices if config.capabilities.special.no_filename

# METADATA
# description: once prepared, aggregate_keys fetched from storage
# scope: document
default aggregate_keys := {}

aggregate_keys := data.internal.prepared.aggregate_keys
