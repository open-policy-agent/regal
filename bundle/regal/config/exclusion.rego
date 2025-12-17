package regal.config

# METADATA
# description: |
#   determines if file should be excluded, either because of an override,
#   or because the specific rule configuration excludes it
#
#   imitates .gitignore pattern matching as best it can
#   ref: https://git-scm.com/docs/gitignore#_pattern_format
excluded_file(category, title, file) if {
	some compiled in _patterns_compiler(rules[category][title].ignore.files)
	glob.match(compiled, ["/"], file)
}

# METADATA
# description: determines if file is ignored globally, via an override or the configuration
ignored_globally(file) if {
	some compiled in _global_ignore_patterns
	glob.match(compiled, ["/"], file)
}

_global_ignore_patterns := compiled if {
	compiled := _patterns_compiler(_params.ignore_files)
} else := _patterns_compiler(merged_config.ignore.files)

# pattern_compiler transforms a glob pattern into a set of glob
# patterns to make the combined set behave as .gitignore
_patterns_compiler(patterns) := {pat |
	some pattern in patterns
	some p in _leading_doublestar_pattern(trim_prefix(_internal_slashes(pattern), "/"))
	some pat in _trailing_slash(p)
} if {
	count(patterns) > 0
}

# Internal slashes means that the path is relative to root,
# if not it can appear anywhere in the hierarchy
#
# myfiledir and mydir/ turns into **/myfiledir and **/mydir/
# mydir/p and mydir/d/ are returned as is
_internal_slashes(pattern) := pattern if {
	contains(trim_suffix(pattern, "/"), "/")
} else := $"**/{pattern}"

# **/pattern might match my/dir/pattern and pattern
# So we branch it into itself and one with the leading **/ removed
_leading_doublestar_pattern(pattern) := {pattern, p} if {
	startswith(pattern, "**/")
	p := substring(pattern, 3, -1)
} else := {pattern}

# If a pattern does not end with a "/", then it can both
# - match a folder => pattern + "/**"
# - match a file => pattern
_trailing_slash(pattern) := {pattern, $"{pattern}/**"} if {
	not endswith(pattern, "/")
	not endswith(pattern, "**")
} else := {$"{pattern}**"} if {
	endswith(pattern, "/")
} else := {pattern}
