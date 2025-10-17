# METADATA
# description: |
#   Highlights text in document depending on cursor position. Currently highlights:
#     - metadata attributes at cursor position
#     - function arguments and references to them in the function head and body
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentHighlight
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.documenthighlight
package regal.lsp.documenthighlight

import data.regal.ast
import data.regal.lsp.completion.location
import data.regal.lsp.util.location as uloc
import data.regal.util

# METADATA
# entrypoint: true
result["response"] := items

# METADATA
# description: Highlights a function args in position
items contains item if {
	[arg, _] := _arg_at_position

	item := {
		"range": uloc.to_range(util.to_location_object(arg.location)),
		"kind": 2, # Write
	}
}

# METADATA
# description: Highlights function arg references in function body when clicked
items contains item if {
	[arg, i] := _arg_at_position

	some expr in ast.found.expressions[sprintf("%d", [i])]

	walk(expr, [_, value])

	value.type == "var"
	value.value == arg.value

	item := {
		"range": uloc.to_range(util.to_location_object(value.location)),
		"kind": 3, # Read
	}
}

# METADATA
# description: Highlights function arg references in head value body when clicked
items contains item if {
	[arg, i] := _arg_at_position

	walk(data.workspace.parsed[input.params.textDocument.uri].rules[i].head.value, [_, value])

	value.type == "var"
	value.value == arg.value

	item := {
		"range": uloc.to_range(util.to_location_object(value.location)),
		"kind": 3, # Read
	}
}

# METADATA
# description: Highlights METADATA itself when clicked
items contains item if {
	startswith(input.regal.file.lines[input.params.position.line], "# METADATA")

	item := {
		"range": {
			"start": {"line": input.params.position.line, "character": 2},
			"end": {"line": input.params.position.line, "character": 10},
		},
		"kind": 1,
	}
}

# METADATA
# description: Highlights all metadata attributes when METADATA header is clicked
items contains item if {
	startswith(input.regal.file.lines[input.params.position.line], "# METADATA")

	module := data.workspace.parsed[input.params.textDocument.uri]
	annotation := _find_annotation(module, input.params.position.line + 1)

	# the annotation attributes have no individual location, so
	# we'll have to find their location in the file from text
	loc := util.to_location_object(annotation.location)

	some i in numbers.range(loc.row, loc.end.row - 1)

	word := _attribute_from_text(input.regal.file.lines[i])

	item := {
		"range": {
			"start": {"line": i, "character": 2},
			"end": {"line": i, "character": 2 + count(word)},
		},
		"kind": 1,
	}
}

# METADATA
# description: Highlights individual metadata attributes when clicked
items contains item if {
	line := input.params.position.line
	word := _attribute_from_text(input.regal.file.lines[line])
	item := {
		"range": {
			"start": {"line": line, "character": 2},
			"end": {"line": line, "character": 2 + count(word)},
		},
		"kind": 1,
	}
}

_arg_at_position := [arg, i] if {
	text := input.regal.file.lines[input.params.position.line]
	word := location.word_at(text, input.params.position.character)

	some i, rule in data.workspace.parsed[input.params.textDocument.uri].rules
	some arg in rule.head.args

	arg.type == "var"
	arg.value == word.text

	loc := util.to_location_object(arg.location)

	input.params.position.line + 1 == loc.row
	input.params.position.character >= loc.col - 1
	input.params.position.character <= (loc.col - 1) + count(arg.value)
}

_find_annotation(module, row) := annotation if {
	util.to_location_row(module.package.annotations[0].location) == row

	annotation := module.package.annotations[0]
}

_find_annotation(module, row) := annotation if {
	annotation := module.rules[_].annotations[_]

	util.to_location_row(annotation.location) == row
}

_attribute_from_text(line) := word if {
	strings.any_prefix_match(line, {
		"# scope:",
		"# title:",
		"# description:",
		"# related_resources:",
		"# authors:",
		"# organizations:",
		"# schemas:",
		"# entrypoint:",
		"# custom:",
	})

	idx := indexof(line, ":")
	idx != -1

	# Trim the leading '# ' and anything following (and including) ':'
	word := substring(line, 2, idx - 2)
}
