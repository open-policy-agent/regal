# METADATA
# description: |
#   Highlights text in document depending on cursor position. Currently highlights:
#     - metadata attributes at cursor position
#     - the references in this file of the symbol at cursor position
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentHighlight
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.documenthighlight

import data.regal.lsp.util.references
import data.regal.util

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := items if items != set()

# METADATA
# description: Highlights the references in this file of the symbol at the cursor
items contains item if {
	some ref in references.refs

	ref.uri == input.params.textDocument.uri

	item := {
		"range": ref.range,
		"kind": ref.kind,
	}
}

# METADATA
# description: Highlights METADATA itself when clicked
items contains item if {
	startswith(input.regal.file.lines[input.params.position.line], "# METADATA")

	item := {
		"range": {
			"start": {
				"line": input.params.position.line,
				"character": 2,
			},
			"end": {
				"line": input.params.position.line,
				"character": 10,
			},
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
			"start": {
				"line": i,
				"character": 2,
			},
			"end": {
				"line": i,
				"character": 2 + count(word),
			},
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
			"start": {
				"line": line,
				"character": 2,
			},
			"end": {
				"line": line,
				"character": 2 + count(word),
			},
		},
		"kind": 1,
	}
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

	i := indexof(line, ":")
	i != -1

	# Trim the leading '# ' and anything following (and including) ':'
	word := substring(line, 2, i - 2)
}
