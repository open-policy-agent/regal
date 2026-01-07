# METADATA
# description: provides completion suggestions for the `default` keyword where applicable
package regal.lsp.completion.providers.default

import data.regal.ast

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: all completion suggestions for default keyword
# scope: document
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	startswith("default", line)

	item := {
		"label": "default",
		"kind": kind.keyword,
		"detail": "default <rule-name> := <value>",
		"textEdit": {
			"range": location.from_start_of_line_to_position(input.params.position),
			"newText": "default ",
		},
	}
}

items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	startswith("default", line)

	some name in ast.rule_and_function_names

	item := {
		"label": $"default {name} := <value>",
		"kind": kind.keyword,
		"detail": $"add default assignment for {name} rule",
		"textEdit": {
			"range": location.from_start_of_line_to_position(input.params.position),
			"newText": $"default {name} := ",
		},
	}
}
