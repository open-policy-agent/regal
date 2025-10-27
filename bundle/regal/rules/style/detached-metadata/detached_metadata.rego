# METADATA
# description: Detached metadata annotation
package regal.rules.style["detached-metadata"]

import data.regal.ast
import data.regal.result
import data.regal.util

report contains violation if {
	some i, block in ast.comments.blocks

	regex.match(`^\s*METADATA`, block[0].text)

	last_row := util.to_location_object(regal.last(block).location).row

	# no need to +1 the index here as rows start counting from 1
	trim_space(input.regal.file.lines[last_row]) == ""

	not _allow_detached(last_row, i, ast.comments.blocks, input.regal.file.lines)

	violation := result.fail(rego.metadata.chain(), result.location(block[0]))
}

# detached metadata is allowed only if another metadata block follows
# directly after the metadata block
_allow_detached(last_row, i, blocks, lines) if {
	next_block_start := blocks[i + 1][0]

	regex.match(`^\s*METADATA`, next_block_start.text)

	next_block_row := util.to_location_object(next_block_start.location).row
	lines_between := array.slice(lines, last_row, next_block_row - 1)

	every line in lines_between {
		line == ""
	}
}
