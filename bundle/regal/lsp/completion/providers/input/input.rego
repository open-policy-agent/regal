# METADATA
# description: provides completion suggestions for the `input` keyword where applicable
package regal.lsp.completion.providers.input

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: all completion suggestions for the input keyword
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	line != ""
	location.in_rule_body(line)

	word := location.word_at(line, input.params.position.character + 1)

	startswith("input", word.text)

	# input, as we mean it here, cannot follow dot (like data.input or input.input)
	before := trim_suffix(line, word.text)
	not endswith(before, ".")

	item := {
		"label": "input",
		"kind": kind.keyword,
		"detail": "input document",
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": "input",
		},
		"documentation": {
			"kind": "markdown",
			"value": _doc,
		},
	}
}

_doc := `# input

'input' refers to the input document being evaluated.
It is a special keyword that allows you to access the data sent to OPA at evaluation time.

To see more examples of how to use 'input', check out the
[policy language documentation](https://www.openpolicyagent.org/docs/policy-language/).

You can also experiment with input in the [Rego Playground](https://play.openpolicyagent.org/).
`
