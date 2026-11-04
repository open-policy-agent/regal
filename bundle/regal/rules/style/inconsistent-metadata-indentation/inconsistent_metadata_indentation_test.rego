package regal.rules.style["inconsistent-metadata-indentation_test"]

import data.regal.ast
import data.regal.config
import data.regal.rules.style["inconsistent-metadata-indentation"] as rule

test_fail_inconsistent_indentation if {
	# The default step is two spaces, so the block indented by one is reported.
	violations := rule.report with input as ast.policy(`
# METADATA
# description: |
#   first block
a := 1

# METADATA
# description: |
#  second block
b := 2
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "#  second block"
	violation.title == "inconsistent-metadata-indentation"
	violation.level == "error"
}

test_fail_configured_indent_num_spaces if {
	# Configured to three, the block indented by one is still the one reported.
	violations := rule.report with input as ast.policy(`
# METADATA
# description: |
#    first block
a := 1

# METADATA
# description: |
#  second block
b := 2
`)
		with config.rules as {"style": {"inconsistent-metadata-indentation": {"indent-num-spaces": 3}}}

	count(violations) == 1
	some violation in violations
	violation.location.text == "#  second block"
}

test_pass_configured_indent_num_spaces if {
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#    first block
a := 1

# METADATA
# description: |
#    second block
b := 2
`)
		with config.rules as {"style": {"inconsistent-metadata-indentation": {"indent-num-spaces": 3}}}

	count(r) == 0
}

test_fail_default_step_when_content_is_indented_by_one if {
	# No configuration: one space of indentation is not the two the rule expects, and a file
	# that consistently indents by one is reported on its first block only.
	violations := rule.report with input as ast.policy(`
# METADATA
# description: |
#  only block in the file
a := 1
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "#  only block in the file"
}

test_pass_consistent_indentation if {
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   first block
a := 1

# METADATA
# description: |
#   second block
b := 2
`)

	count(r) == 0
}

test_pass_single_metadata_block if {
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   only block in the file
a := 1
`)

	count(r) == 0
}

test_pass_no_multiline_metadata if {
	r := rule.report with input as ast.policy(`
# METADATA
# description: single line
a := 1

# METADATA
# description: also a single line
b := 2
`)

	count(r) == 0
}

test_fail_middle_block_differs if {
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   first block
a := 1

# METADATA
# description: |
#    middle block
b := 2

# METADATA
# description: |
#   last block
c := 3
`)

	count(r) == 1
}

test_pass_nested_structure_is_not_content if {
	# `related_resources` items are nested a level deeper than the string content,
	# which is not the indentation being compared here.
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   first block
# related_resources:
#   - description: docs
#     ref: https://example.com
a := 1

# METADATA
# description: |
#   second block
# related_resources:
#   - description: docs
#     ref: https://example.com
b := 2
`)

	count(r) == 0
}

test_fail_nested_string_indented_differently if {
	# A nested string — the description of a related_resources item — is compared like any
	# other, from the leading spaces of its own key line: its content is measured against the
	# configured step, not against the content of the block's top-level strings. So a nested
	# string indented deeper than the step is reported even when the file agrees on it.
	violations := rule.report with input as ast.policy(`
# METADATA
# description: |
#   first block
# related_resources:
#   - description: |
#       docs
#     ref: https://example.com
a := 1

# METADATA
# description: |
#   second block
# related_resources:
#   - description: |
#           also docs but indented differently in nested attribute
#     ref: https://example.com
b := 2
`)

	count(violations) == 2

	sort([violation.location.text | some violation in violations]) == [
		"#           also docs but indented differently in nested attribute",
		"#       docs",
	]
}
