# METADATA
# description: |
#   the `regov1`` provider provides completion suggestions for
#   `rego.v1` following an `import` declaration
package regal.lsp.completion.providers.regov1

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: completion suggestion for rego.v1 import
items contains item if {
	input.regal.file.rego_version != "v1" # the rego.v1 import is not used in v1 Rego
	not strings.any_prefix_match(input.regal.file.lines, "import rego.v1")

	line := input.regal.file.lines[input.params.position.line]

	startswith(line, "import ")

	word := location.ref_at(line, input.params.position.character + 1)

	startswith("rego.v1", word.text)

	item := {
		"label": "rego.v1",
		"kind": kind.module,
		"detail": "use rego.v1",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": "rego.v1\n\n",
		},
	}
}
