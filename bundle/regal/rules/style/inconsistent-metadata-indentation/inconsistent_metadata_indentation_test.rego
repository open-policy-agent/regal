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

test_fail_array_items_indented_with_one_space if {
	# The file mixes both valid shapes: the array under `authors` is indented by one
	# space and the one under `related_resources` by the two the rule expects, so the
	# first is the one reported.
	violations := rule.report with input as ast.policy(`
# METADATA
# authors:
# - anders
a := 1

# METADATA
# related_resources:
#   - https://example.org
b := 2
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "# - anders"
}

test_fail_array_items_indented_with_four_spaces if {
	violations := rule.report with input as ast.policy(`
# METADATA
# authors:
#     - anders
a := 1
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "#     - anders"
}

test_pass_array_items_indented_with_two_spaces if {
	r := rule.report with input as ast.policy(`
# METADATA
# authors:
#   - anders
# related_resources:
#   - ref: https://example.org
a := 1
`)

	count(r) == 0
}

test_fail_configured_array_indent_num_spaces if {
	violations := rule.report with input as ast.policy(`
# METADATA
# authors:
#   - anders
a := 1
`)
		with config.rules as {"style": {"inconsistent-metadata-indentation": {"array-indent-num-spaces": 4}}}

	count(violations) == 1
	some violation in violations
	violation.location.text == "#   - anders"
}

test_pass_configured_array_indent_num_spaces if {
	r := rule.report with input as ast.policy(`
# METADATA
# authors:
#     - anders
a := 1
`)
		with config.rules as {"style": {"inconsistent-metadata-indentation": {"array-indent-num-spaces": 4}}}

	count(r) == 0
}

test_pass_array_item_inside_a_block_scalar_is_text if {
	# A line starting with "- " inside a multi-line string is content, not a list item, even
	# when the content above it ends with a colon and so looks like the key of a list.
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   Steps:
#   - first
a := 1
`)

	count(r) == 0
}

test_fail_one_violation_per_block_for_array_items if {
	# Three deviating items in one block are reported once, at the first.
	violations := rule.report with input as ast.policy(`
# METADATA
# authors:
# - anders
# - someone
# - else
a := 1
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "# - anders"
}

test_pass_content_line_ending_with_a_colon_does_not_parent_the_next_item if {
	# `Note:` is content of the first item's description and ends with a colon, but it is
	# indented deeper than the item below, so it cannot be the key that item belongs to:
	# that is still `related_resources`.
	r := rule.report with input as ast.policy(`
# METADATA
# related_resources:
#   - ref: https://example.org/a
#     description: |
#       Note:
#   - ref: https://example.org/b
a := 1
`)

	count(r) == 0
}

test_pass_blank_line_inside_a_block_scalar if {
	# A blank comment line is part of the multi-line string, so the `- ` line after it
	# is text too, not an item.
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   Steps:
#
#   - first
a := 1
`)

	count(r) == 0
}

test_pass_nested_array_uses_its_own_key if {
	r := rule.report with input as ast.policy(`
# METADATA
# custom:
#   list:
#     - a
a := 1
`)

	count(r) == 0
}

test_fail_nested_array_uses_its_own_key if {
	violations := rule.report with input as ast.policy(`
# METADATA
# custom:
#   list:
#   - a
a := 1
`)

	count(violations) == 1
	some violation in violations
	violation.location.text == "#   - a"
}

test_pass_array_after_a_block_scalar if {
	# The string ends where a line falls back to the key's indentation, so the array
	# below it is real and is indented the way the rule expects.
	r := rule.report with input as ast.policy(`
# METADATA
# description: |
#   - not an item
# authors:
#   - anders
a := 1
`)

	count(r) == 0
}
