# METADATA
# description: Multi-line metadata should be indented consistently across the metadata blocks of a file.
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

	regex.match(`:\s*[|>][-+]?\s*$`, block[i - 1].text)

	step := leading_spaces(comment.text) - leading_spaces(block[i - 1].text)
	step > 0

	row := comment.row
]

# METADATA
# description: the number of spaces between the leading "#" and the text of a comment
leading_spaces(text) := count(regex.find_n(`^#\s*`, text, 1)[0]) - 1
