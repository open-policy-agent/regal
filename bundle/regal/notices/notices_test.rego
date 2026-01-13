package regal.notices_test

import data.regal.notices

test_promoted_notices_with_level_error if {
	result := notices.promoted_notices with data.regal.rules.imports["use-rego-v1"].notices as {_example_notice}
		with data.internal.user_config as {"rules": {"imports": {"use-rego-v1": {"level": "error"}}}}

	# With user config level set to error, the severity should be promoted to error
	result.imports["use-rego-v1"] == {object.union(_example_notice, {"severity": "error"})}
}

test_promoted_notices_with_level_ignore if {
	result := notices.promoted_notices with data.regal.rules.imports["use-rego-v1"].notices as {_example_notice}
		with data.internal.user_config as {"rules": {"imports": {"use-rego-v1": {"level": "ignore"}}}}

	# With user config level set to ignore, the severity should stay none
	result.imports["use-rego-v1"] == {_example_notice}
}

test_promoted_notices_with_level_warning if {
	result := notices.promoted_notices with data.regal.rules.imports["use-rego-v1"].notices as {_example_notice}
		with data.internal.user_config as {"rules": {"imports": {"use-rego-v1": {"level": "warning"}}}}

	# With user config level set to warning, the severity should be promoted to warning
	result.imports["use-rego-v1"] == {object.union(_example_notice, {"severity": "warning"})}
}

test_promoted_notices_configured_without_level if {
	# Rule is configured but level field is not present
	result := notices.promoted_notices with data.regal.rules.imports["use-rego-v1"].notices as {_example_notice}
		with data.internal.user_config as {"rules": {"imports": {"use-rego-v1": {}}}}

	# When configured without level, should default to error
	result.imports["use-rego-v1"] == {object.union(_example_notice, {"severity": "error"})}
}

test_promoted_notices_mixed_configured_and_unconfigured if {
	notice_configured_rule := {
		"category": "imports",
		"description": "Configured rule",
		"level": "notice",
		"title": "use-rego-v1",
		"severity": "none",
	}

	notice_unconfigured_rule := {
		"category": "bugs",
		"description": "Unconfigured rule",
		"level": "notice",
		"title": "deprecated-builtin",
		"severity": "none",
	}

	result := notices.promoted_notices with data.regal.rules.imports["use-rego-v1"].notices as {notice_configured_rule}
		with data.regal.rules.bugs["deprecated-builtin"].notices as {notice_unconfigured_rule}
		with data.internal.user_config as {"rules": {"imports": {"use-rego-v1": {"level": "error"}}}}

	# Configured rule should have severity promoted to error
	result.imports["use-rego-v1"] == {object.union(notice_configured_rule, {"severity": "error"})}

	# Unconfigured rule should keep original severity: none
	result.bugs["deprecated-builtin"] == {notice_unconfigured_rule}
}

_example_notice := {
	"category": "imports",
	"description": "Test rule description",
	"level": "notice",
	"title": "use-rego-v1",
	"severity": "none",
}
