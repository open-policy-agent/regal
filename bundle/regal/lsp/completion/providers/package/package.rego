# METADATA
# description: provides completion suggestions for the `package` keyword where applicable
package regal.lsp.completion.providers.package

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: completion suggestions for package keyword
items contains item if {
	not strings.any_prefix_match(input.regal.file.lines, "package ")

	startswith("package", input.regal.file.lines[input.params.position.line])

	item := {
		"label": "package",
		"kind": kind.keyword,
		"detail": "package <package-name>",
		"textEdit": {
			"range": location.from_start_of_line_to_position(input.params.position),
			"newText": "package ",
		},
	}
}
