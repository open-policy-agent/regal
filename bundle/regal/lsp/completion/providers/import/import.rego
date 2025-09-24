# METADATA
# description: provides completion suggestions for the `import` keyword where applicable
package regal.lsp.completion.providers.import

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: all completion suggestions for the import keyword
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	startswith("import", line)

	word := location.word_at(line, input.params.position.character + 1)

	item := {
		"label": "import",
		"kind": kind.keyword,
		"detail": "import <path>",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": "import ",
		},
	}
}
