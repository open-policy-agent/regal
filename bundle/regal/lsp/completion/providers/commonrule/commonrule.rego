# METADATA
# description: |
#   provides completions for common rule names, like 'allow' or 'deny'
package regal.lsp.completion.providers.commonrule

import data.regal.lsp.completion.kind
import data.regal.lsp.location

_suggested_names := {
	"allow",
	"authorized",
	"deny",
}

# METADATA
# description: all completion suggestions for common rule names
items contains item if {
	line := input.regal.file.lines[input.params.position.line]
	not startswith(line, "default")

	some label in _suggested_names

	startswith(label, line)

	item := {
		"label": label,
		"kind": kind.snippet,
		"detail": "common rule name",
		"documentation": {
			"kind": "markdown",
			"value": $`"{label}" is a common rule name`,
		},
		"textEdit": {
			"range": location.from_start_of_line_to_position(input.params.position),
			"newText": $"{label} ",
		},
	}
}

# METADATA
# description: all completion suggestions for common rule names
items contains item if {
	line := input.regal.file.lines[input.params.position.line]
	startswith(line, "default ")

	word := location.word_at(line, input.params.position.character + 1)

	some label in _suggested_names

	startswith(label, word.text)

	item := {
		"label": label,
		"kind": kind.snippet,
		"detail": "common rule name",
		"documentation": {
			"kind": "markdown",
			"value": $`"{label}" is a common rule name`,
		},
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": label,
		},
	}
}
