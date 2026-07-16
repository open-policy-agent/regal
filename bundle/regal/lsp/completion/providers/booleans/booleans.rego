# METADATA
# description: the boolean provider suggests `true`/`false` values where appropriate
package regal.lsp.completion.providers.booleans

import data.regal.lsp.client
import data.regal.lsp.completion.kind
import data.regal.lsp.location

# METADATA
# description: completion suggestions for true/false boolean values
items contains item if {
	line := input.regal.file.lines[input.params.position.line]
	line != ""

	word := location.word_at(line, input.params.position.character + 1)
	word.text != ""

	item := _bool_suggestion(_matched(word), word, client.supports.edit_range_defaults)
}

_bool_str(s) := "true" if startswith("true", s)
_bool_str(s) := "false" if startswith("false", s)

_matched(word) := _bool_str(word.text) if regex.match(`^\s+$`, word.text_before)
_matched(word) := _bool_str(word.text) if strings.any_suffix_match(trim_space(word.text_before), [
	"if",
	",",
	":",
	"(",
	"[",
	"=", # covers =, ==, !=, :=
])

_bool_suggestion(bool, _, true) := {
	"label": bool,
	"labelDetails": {
		"description": "value",
	},
	"kind": kind.value,
	"detail": "boolean value",
}

_bool_suggestion(bool, word, false) := {
	"label": bool,
	"labelDetails": {
		"description": "value",
	},
	"kind": kind.value,
	"detail": "boolean value",
	"textEdit": {
		"range": location.word_range(word, input.params.position),
		"newText": bool,
	},
}
