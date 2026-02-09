# METADATA
# description: Max rule length exceeded
package regal.rules.style["rule-length"]

import data.regal.config
import data.regal.result
import data.regal.util

report contains violation if {
	cfg := config.rules.style["rule-length"]

	some rule in input.rules

	is_test := startswith(rule.head.ref[0].value, "test_")
	max_length := cfg[{false: "max-rule-length", true: "max-test-rule-length"}[is_test]]

	text := util.to_location_object(rule.location).text

	# cheaper check first, and only account for comments if too long to begin with
	rule_length := strings.count(text, "\n") + 1
	rule_length > max_length

	_line_count(cfg, text, rule_length) > max_length

	not _no_body_exception(cfg, rule)

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

_no_body_exception(cfg, rule) if {
	cfg["except-empty-body"] == true
	not rule.body
}

_line_count(cfg, _, rule_length) := rule_length if cfg["count-comments"] == true

_line_count(cfg, text, rule_length) := without_comments if {
	not cfg["count-comments"]

	without_comments := rule_length - count([1 |
		some line in split(text, "\n")
		startswith(trim_space(line), "#")
	])
}
