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

	some original_notice in notices

	not config.user_config.rules[category][title]
}

promoted_notices[category][title] contains notice if {
	some category, title
	notices := data.regal.rules[category][title].notices

	some notice in notices

	rule_config := config.user_config.rules[category][title]
	object.get(rule_config, "level", "") == "ignore"
}

promoted_notices[category][title] contains notice if {
	some category, title
	notices := data.regal.rules[category][title].notices

	some original_notice in notices

	rule_config := config.user_config.rules[category][title]
	object.get(rule_config, "level", "") != "ignore"

	# Use configured level as severity, or default to "error"
	new_severity := object.get(rule_config, "level", "error")

	notice := object.union(original_notice, {"severity": new_severity})
}
