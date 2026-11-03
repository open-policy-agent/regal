# METADATA
# description: Inconsistent indentation of multi-line metadata and array items
# related_resources:
#   - description: documentation
#     ref: https://www.openpolicyagent.org/projects/regal/rules/style/inconsistent-metadata-indentation
package regal.rules.style["inconsistent-metadata-indentation"]

import data.regal.ast
import data.regal.config
import data.regal.result

# METADATA
# description: reports metadata blocks whose multi-line strings are indented differently than the configured step
report contains violation if {
	some block in ast.comments.blocks
	regex.match(`^#\s*METADATA\s*$`, block[0].text)

	# One violation per block, at the first line that deviates.
	deviating := sort([row | some [row, step] in scalar_lines(block); step != expected_step])

	some comment in block
	comment.row == deviating[0]

	violation := result.fail(rego.metadata.chain(), result.with_text(comment))
}

# METADATA
# description: reports metadata blocks whose array items are indented differently than the configured step
report contains violation if {
	some block in ast.comments.blocks
	regex.match(`^#\s*METADATA\s*$`, block[0].text)

	# One violation per block, at the first item that deviates.
	deviating := sort([row | some [row, step] in array_items(block); step != expected_array_step])

	some comment in block
	comment.row == deviating[0]

	violation := result.fail(rego.metadata.chain(), result.with_text(comment))
}

# METADATA
# description: the indentation step a multi-line string should use
expected_step := object.get(config.rules, ["style", "inconsistent-metadata-indentation", "indent-num-spaces"], 2)

# METADATA
# description: the indentation step an array item should use
expected_array_step := object.get(
	config.rules,
	["style", "inconsistent-metadata-indentation", "array-indent-num-spaces"],
	2,
)

# METADATA
# description: |
#   the multi-line strings of a metadata block, with the indentation step used
#   by their first line, i.e. how much deeper the content is indented than the
#   key it belongs to
scalar_lines(block) := [[row, step] |
	some i, comment in block
	i > 0

	# only a line ending in a "|" or ">" block scalar (optionally with a "-" or "+" chomping
	# indicator) opens a multi-line string, and only those have indented content to compare
	regex.match(`:\s*[|>][-+]?\s*$`, block[i - 1].text)

	# the leading "#" is counted on both sides and cancels out below
	comment_indent := count(comment.text) - count(trim_left(comment.text, "# "))
	key_indent := count(block[i - 1].text) - count(trim_left(block[i - 1].text, "# "))

	step := comment_indent - key_indent
	step > 0

	row := comment.row
]

# METADATA
# description: |
#   the items of the arrays in a metadata block, with the indentation step each
#   uses relative to the key that opens the list
array_items(block) := [[row, step] |
	some i, comment in block
	i > 0
	array_item(comment.text)

	# a "- " inside a multi-line string is text, not an item of any array
	not scalar_content(block, i)

	step := indent(comment.text) - indent(block[key_index(block, i)].text)
	row := comment.row
]

# METADATA
# description: an item of a metadata array
array_item(text) if regex.match(`^#\s*-\s`, text)

# METADATA
# description: a line that opens a value with a colon, and is not an array item
key_line(text) if regex.match(`^#\s*[^-\s].*:\s*$`, text)

# METADATA
# description: |
#   the index of the key line the item at index i belongs to: the nearest key
#   above it that is not indented deeper, so a key nested inside a multi-line
#   string cannot become the parent of a list that follows it
key_index(block, i) := max({j |
	some j
	j < i
	key_line(block[j].text)
	indent(block[j].text) <= indent(block[i].text)
})

# METADATA
# description: |
#   whether the line at index i is the content of a block scalar, i.e. it follows a
#   line opening one and does not fall back to that line's indentation
scalar_content(block, i) if {
	some j
	j < i
	regex.match(`:\s*[|>][-+]?\s*$`, block[j].text)

	every m in numbers.range(j + 1, i) {
		inside_scalar(block, j, m)
	}
}

# METADATA
# description: |
#   whether a line is still part of the block scalar opened at index j: a blank
#   comment line says nothing about the block's extent, and any other line has to
#   stay deeper than the line that opened it
inside_scalar(block, _, m) if trim_left(block[m].text, "# ") == ""

# METADATA
# description: whether a line stays deeper than the line that opened the block scalar
inside_scalar(block, j, m) if indent(block[m].text) > indent(block[j].text)

# METADATA
# description: |
#   how far the text of a commented line is indented: the characters before the
#   first one that is neither "#" nor a space, so that a key and an item are
#   comparable even when the item is written as "# - value"
indent(text) := indexof(text, trim_left(text, "# "))
