# METADATA
# description: the boolean provider suggests `true`/`false` values where appropriate
package regal.lsp.completion.providers.booleans

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: completion suggestions for true/false
items contains item if {
	line := input.regal.file.lines[input.params.position.line]
	line != ""

	words := regex.split(`\s+`, line)

	words_on_line := count(words)
	previous_word := words[words_on_line - 2]

	previous_word in {"==", ":="}

	word := location.word_at(line, input.params.position.character + 1)

	some b in ["true", "false"]

	startswith(b, word.text)

	item := {
		"label": b,
		"kind": kind.constant,
		"detail": "boolean value",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": b,
		},
	}
}
