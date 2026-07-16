# METADATA
# description: |
#   the `future` provider provides completion suggestions for
#   future keywords following an `import` declaration, provided
#   that any are found in the determined capabilities
package regal.lsp.completion.providers.future

import data.regal.lsp.completion.kind
import data.regal.lsp.location

# METADATA
# description: completion suggestion for future import
items contains item if {
	keywords := data.workspace.config.capabilities.future_keywords
	keywords != []

	line := input.regal.file.lines[input.params.position.line]
	startswith(line, "import ")

	word := location.ref_at(line, input.params.position.character + 1)

	some keyword in keywords

	ref := $"future.keywords.{keyword}"
	startswith(ref, word.text)

	item := _item(keyword, ref, word, input.params.position)
}

_item(keyword, ref, word, position) := {
	"label": ref,
	"labelDetails": {
		"description": "reference",
	},
	"kind": kind.reference,
	"detail": $"import future `{keyword}` keyword",
	"textEdit": {
		"range": location.word_range(word, position),
		"newText": ref,
	},
}
