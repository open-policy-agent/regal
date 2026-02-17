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
