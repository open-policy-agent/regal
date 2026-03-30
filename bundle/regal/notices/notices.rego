# METADATA
# description: |
#   package with functionality for post-processing notices
#   to ensure they are presented correctly as errors when relevant.
package regal.notices

import data.regal.config

# METADATA
# scope: rule
# description: |
#   promoted_notices maps notices from rules, potentially changing their severity
#   based on user configuration
promoted_notices[category][title] contains original_notice if {
	some category, title
	notices := data.regal.rules[category][title].notices

	not config.user_config.rules[category][title]

	some original_notice in notices
}

promoted_notices[category][title] contains notice if {
	some category, title
	config.user_config.rules[category][title].level == "ignore"

	some notice in data.regal.rules[category][title].notices
}

promoted_notices[category][title] contains object.union(notice, severity) if {
	some category, title
	rule_config := config.user_config.rules[category][title]

	# Use configured level as severity, or default to "error"
	level := object.get(rule_config, "level", "error")
	level != "ignore"

	severity := {"severity": level}

	some notice in data.regal.rules[category][title].notices
}
