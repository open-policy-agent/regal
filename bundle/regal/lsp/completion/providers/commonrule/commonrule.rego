# METADATA
# description: |
#   provides completions for common rule names, like 'allow' or 'deny'
package regal.lsp.completion.providers.commonrule

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

_suggested_names := {
	"allow",
	"authorized",
	"deny",
}

# METADATA
# description: all completion suggestions for common rule names
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	some label in _suggested_names

	startswith(label, line)

	item := {
		"label": label,
		"kind": kind.snippet,
		"detail": "common name",
		"documentation": {
			"kind": "markdown",
			"value": sprintf("%q is a common rule name", [label]),
		},
		"textEdit": {
			"range": location.from_start_of_line_to_position(input.params.position),
			"newText": concat("", [label, " "]),
		},
	}
}
