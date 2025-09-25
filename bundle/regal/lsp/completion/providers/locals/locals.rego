# METADATA
# description: provides completion suggestions for local symbols in scope
package regal.lsp.completion.providers.locals

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: completion suggestions for local symbols
items contains item if {
	line := input.regal.file.lines[input.params.position.line]
	line != ""

	location.in_rule_body(line)
	not _excluded(line, input.params.position)

	word := location.word_at(line, input.params.position.character + 1)

	not endswith(word.text_before, ".")

	some local in location.find_locals(data.workspace.parsed[input.params.textDocument.uri].rules, {
		"row": input.params.position.line + 1,
		"col": input.params.position.character + 1,
	})

	startswith(local, word.text)

	not local in _same_line_loop_vars(line)

	item := {
		"label": local,
		"kind": kind.variable,
		"detail": "local variable",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": local,
		},
	}
}

# exclude local suggestions in function args definition,
# as those would recursively contribute to themselves
# regal ignore:narrow-argument
_excluded(line, position) if _function_args_position(substring(line, 0, position.character))

_function_args_position(text) if {
	contains(text, "(")
	not contains(text, "=")
	text == trim_left(text, " \t")
}

default _same_line_loop_vars(_) := []

_same_line_loop_vars(line) := vars if {
	regex.match(`^\s*(some|every)`, line)

	vars := split(regex.replace(line, `(?:\s*(?:some|every)\s+)(\w+)(?:,?\s*)(\w+)?(?:\s+.*)in.*`, "$1,$2"), ",")
}
