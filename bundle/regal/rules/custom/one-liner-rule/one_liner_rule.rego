# METADATA
# description: Rule body could be made a one-liner
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/custom/one-liner-rule
package regal.rules.custom["one-liner-rule"]

import future.keywords.not

import data.regal.ast
import data.regal.capabilities
import data.regal.config
import data.regal.result
import data.regal.util

# METADATA
# description: Missing capability for keyword `if`
# custom:
#   severity: warning
notices contains result.notice(rego.metadata.chain()) if not capabilities.has_if

report contains violation if {
	some rule in input.rules

	# Bail out of rules with else for now. It is possible that they can be made
	# one-liners, but they'll often be longer than the preferred line length
	# We can come back to this later, but for now let's just make this an
	# exception documented for this rule
	not rule.else

	# Single expression in body required for one-liner
	count(rule.body) == 1

	# Note that this will give us the text representation of the whole rule,
	# which we'll need as the "if" is only visible here ¯\_(ツ)_/¯
	rule_location := util.to_location_object(rule.location)
	lines := [trim_space(line) | some line in split(rule_location.text, "\n")]

	regex.match(`\s+if`, lines[0])
	_rule_body_brackets(lines)

	# ideally we'd take style preference into account but for now assume tab == 4 spaces
	# then just add the sum of the line counts minus the removed '{' character
	# redundant parens added by `opa fmt` :/
	((4 + count(lines[0])) + count(lines[1])) - 1 < _max_line_length

	not {
		num_lines := count(lines)

		some location in ast.comments_decoded

		location.row > rule_location.row
		location.row < rule_location.row + num_lines
	}

	violation := result.fail(rego.metadata.chain(), result.location(rule.head))
}

default _max_line_length := 120

_max_line_length := config.rules.custom["one-liner-rule"]["max-line-length"]

# K&R style
_rule_body_brackets(lines) if regex.match(`.*if\s*{$`, lines[0])

# Allman style
_rule_body_brackets(lines) if {
	not endswith(lines[0], "{")
	startswith(lines[1], "{")
}
