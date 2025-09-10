# METADATA
# description: |
#   base modules for working with Regal's configuration in Rego
#   this includes responsibilities like providing capabilities, or
#   to determine which rules to enable/disable, and what files to
#   ignore
package regal.config

# METADATA
# description: the path prefix value set on the current linter instance
# scope: document
default path_prefix := ""

path_prefix := data.internal.path_prefix

# METADATA
# description: the base URL for documentation of linter rules
docs["base_url"] := "https://www.openpolicyagent.org/projects/regal/rules"

# METADATA
# description: returns the canonical URL for documentation of the given rule
docs["resolve_url"](url, category) := replace(
	replace(url, "$baseUrl", docs.base_url),
	"$category", category,
)

# METADATA
# description: the default configuration with user config merged on top (if provided)
merged_config := data.internal.combined_config

# ensure that config.rules can be referenced in tests even without mocking
default rules := {}

# METADATA
# description: the merged (default and user) configuration for rules
# scope: document
rules := merged_config.rules

# METADATA
# description: the resolved capabilities sourced from Regal and user configuration
capabilities := object.union(merged_config.capabilities, {"special": _special})

_special contains "no_filename" if input.regal.file.name == "stdin"

default _params := {
	"ignore_files": [],
	"disable_all": false,
	"enable_all": false,
	"disable_category": [],
	"enable_category": [],
	"disable": [],
	"enable": [],
}

_params := data.eval.params

default for_rule(_, _) := {"level": "error"}

# METADATA
# description: |
#   Returns the configuration applied (i.e. the provided configuration
#   merged with any user configuration). Rule authors should normally not
#   consider the `level`` attribute, as whether a rule is evaluated or not
#   is determined by the main policy, based not only on configuration but
#   potentially also overrides. Use `level_for_rule` to determine the
#   exact level as determined during evaluation.
# scope: document
for_rule(category, title) := rules[category][title]

# METADATA
# description: answers whether a rule is ignored in the most efficient way
ignored_rule(category, title) if {
	_force_disabled(_params, category, title)
} else if {
	rules[category][title].level == "ignore"
	not _force_enabled(_params, category, title)
}

# METADATA
# description: returns the level set for rule, based on configuration and possibly overrides
level_for_rule(category, title) := "ignore" if {
	_force_disabled(_params, category, title)
} else := "error" if {
	_force_enabled(_params, category, title)
} else := level if {
	level := rules[category][title].level
} else := "error"

_force_disabled(params, _, title) if title in params.disable

_force_disabled(params, category, title) if {
	params.disable_all == true
	not category in params.enable_category
	not title in params.enable
}

_force_disabled(params, category, title) if {
	category in params.disable_category
	not title in params.enable
}

_force_enabled(params, _, title) if title in params.enable

_force_enabled(params, category, title) if {
	params.enable_all == true
	not category in params.disable_category
	not title in params.disable
}

_force_enabled(params, category, title) if {
	category in params.enable_category
	not title in params.disable
}
