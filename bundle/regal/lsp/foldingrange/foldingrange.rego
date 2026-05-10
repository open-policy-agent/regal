# METADATA
# description: |
#   handler for textDocument/foldingRange requests, supporting folding most
#   AST nodes that may span multiple lines, as well as comment and import "blocks"
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_foldingRange
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.foldingrange

import data.regal.ast
import data.regal.lsp.client

# METADATA
# entrypoint: true
result["response"] := ranges if {
	not client.capabilities.textDocument.foldingRange.rangeLimit
} else := ranges if {
	count(ranges) <= client.capabilities.textDocument.foldingRange.rangeLimit
} else := limited if {
	# NOTE: extremely naive implementation for now as this doesn't feel like it's worth
	# optimizing for, but at least we honor the limit should the client provide one. For
	# reference, VS Code uses a default limit of 5000, so this is likely never going to
	# be hit in practice.
	limited := array.slice(
		[range | some range in ranges],
		0,
		client.capabilities.textDocument.foldingRange.rangeLimit,
	)
}

# METADATA
# description: Comment blocks
ranges contains range if {
	some block in ast.comments.blocks

	count(block) > 1

	# note: the comment locations have already been deserialized in ast.comments.blocks
	range := _block_range(block[0], regal.last(block), only_lines)
}

# METADATA
# description: Imports, lines and characters. Note that we treat all imports as single block.
# scope: rule
ranges contains range if {
	not only_lines

	module := data.workspace.parsed[input.params.textDocument.uri]

	count(module.imports) > 1

	parts_start := split(module.imports[0].location, ":")
	last_term := regal.last(regal.last(module.imports).path.value)
	parts_end := split(last_term.location, ":")

	range := {
		"startLine": to_number(parts_start[0]) - 1,
		"startCharacter": to_number(parts_start[1]) - 1,
		"endLine": to_number(parts_end[0]) - 1,
		"endCharacter": to_number(parts_end[1]) - 1,
		"kind": "imports",
	}
}

# METADATA
# description: Imports, only lines. Note that we treat all imports as single block.
# scope: rule
ranges contains range if {
	only_lines

	module := data.workspace.parsed[input.params.textDocument.uri]

	count(module.imports) > 1

	parts_start := split(module.imports[0].location, ":")
	last_term := regal.last(regal.last(module.imports).path.value)
	parts_end := split(last_term.location, ":")

	range := {
		"startLine": to_number(parts_start[0]) - 1,
		"endLine": to_number(parts_end[0]) - 1,
		"kind": "imports",
	}
}

# METADATA
# description: Rules
ranges contains range if {
	module := data.workspace.parsed[input.params.textDocument.uri]

	some rule in module.rules

	rule_range := _region_range(rule.location, only_lines)

	# TBD else ranges

	sub_ranges := {sub_range |
		# if the rule range is less than 3 lines, there can't be
		# any sub-ranges in it, and we can skip an expensive walk
		rule_range.endLine - rule_range.startLine >= 2

		walk(rule, [_, node])

		node.type in {
			"object",
			"set",
			"array",
			"arraycomprehension",
			"objectcomprehension",
			"setcomprehension",
		}

		sub_range := _region_range(node.location, only_lines)
	}

	some range in ({rule_range} | sub_ranges)
}

# only_lines: true
_region_range(location, true) := fold if {
	[start_line, _, end_line, _] := split(location, ":")

	start_line != end_line

	# note that we set endLine as line _before_ the closing brace/bracket,
	# as that shows the end of the object/set/whatever in the client, which
	# seems to be how other implementations do it as well
	fold := {
		"startLine": to_number(start_line) - 1,
		"endLine": to_number(end_line) - 2,
		"kind": "region",
	}
}

# only_lines: false
_region_range(location, false) := fold if {
	[start_line, start_char, end_line, _] := split(location, ":")

	start_line != end_line

	# see note above about endLine! this makes endCharacter misleading, as it's fetched
	# from the closing brace/bracket on the *actual last line*
	# the spec however states that "if not defined, defaults to the length of the end line"
	# which is exactly what we want.. so simply omit sending endCharacter here
	fold := {
		"startLine": to_number(start_line) - 1,
		"startCharacter": to_number(start_char) - 1,
		"endLine": to_number(end_line) - 2,
		"kind": "region",
	}
}

_block_range(start_location, end_location, true) := {
	"startLine": start_location.row - 1,
	"endLine": end_location.row - 1,
	"kind": "comment",
}

_block_range(start_location, end_location, false) := {
	"startLine": start_location.row - 1,
	"startCharacter": start_location.col - 1,
	"endLine": end_location.row - 1,
	"endCharacter": end_location.end.col - 1,
	"kind": "comment",
}

# METADATA
# description: |
#   clients that that report this capability only support folding entire lines,
#   and we can skip calculating character positions
# scope: document
default only_lines := false

only_lines := client.capabilities.textDocument.foldingRange.lineFoldingOnly
