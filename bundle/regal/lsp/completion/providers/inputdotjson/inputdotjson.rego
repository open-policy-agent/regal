# METADATA
# description: |
#   the `inputdotjson` provider returns suggestions based on the `input.json`
#   data structure (if such a file is found), so that e.g. content like:
#   ```json
#   {
#     "user": {"roles": ["admin"]},
#     "request": {"method": "GEt"}
#   }
#   ```
#   would suggest `input.user`, `input.user.roles`, `input.request`,
#   `input.request.method, and so on
package regal.lsp.completion.providers.inputdotjson

import data.regal.lsp.completion.kind
import data.regal.lsp.location

# METADATA
# description: items contains suggestions from `input.json` in expressions
# scope: rule
items contains item if {
	input.regal.environment.input_path != ""

	line := input.regal.file.lines[input.params.position.line]
	line != ""
	location.in_rule_body(line)

	word := location.ref_at(line, input.params.position.character + 1)
	startswith(word.text, "i")

	edit := location.word_range(word, input.params.position)

	some [suggestion, type] in _input_paths

	startswith(suggestion, word.text)

	item := {
		"label": suggestion,
		"kind": kind.variable,
		"detail": type,
		"documentation": _documentation,
		"textEdit": {
			"range": edit,
			"newText": suggestion,
		},
	}
}

# METADATA
# description: items contains suggestions from `input.json` in imports
# scope: rule
items contains item if {
	input.regal.environment.input_path != ""

	line := input.regal.file.lines[input.params.position.line]
	startswith(line, "import ")

	word := location.ref_at(line, input.params.position.character + 1)
	startswith(word.text, "i")

	edit := location.word_range(word, input.params.position)

	some [suggestion, type] in _input_paths

	startswith(suggestion, word.text)

	item := {
		"label": suggestion,
		"kind": kind.variable,
		"detail": type,
		"documentation": _documentation,
		"textEdit": {
			"range": edit,
			"newText": suggestion,
		},
	}
}

_documentation := {
	"kind": "markdown",
	"value": $"(inferred from [`input.json`]({input.regal.environment.input_path}))",
}

_input_paths contains [$"input.{concat(".", path)}", type_name(value)] if {
	input_doc := data.workspace.inputs[input.regal.environment.input_path]

	walk(input_doc, [path, value])

	path != []

	# don't traverse into arrays
	every value in path {
		is_string(value)
	}
}
