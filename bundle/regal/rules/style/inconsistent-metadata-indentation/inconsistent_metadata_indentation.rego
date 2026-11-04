# METADATA
# description: Inconsistent indentation of multi-line metadata
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
# description: the indentation step a multi-line string should use
expected_step := object.get(config.rules, ["style", "inconsistent-metadata-indentation", "indent-num-spaces"], 2)

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
