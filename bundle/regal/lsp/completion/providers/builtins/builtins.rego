# METADATA
# description: Completion suggestions for built-in functions
package regal.lsp.completion.providers.builtins

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location
import data.regal.lsp.template

# METADATA
# description: suggest built-in functions matching typed ref
items contains item if {
	position := location.to_position(input.regal.context.location)
	line := input.regal.file.lines[position.line]

	line != ""
	not startswith(line, "default ")
	location.in_rule_body(line)

	ref := location.ref_at(line, input.regal.context.location.col)

	some builtin in data.workspace.builtins

	not builtin.infix # avoid suggesting 'eq', 'plus', etc
	not builtin.deprecated

	startswith(builtin.name, ref.text)

	item := {
		"label": builtin.name,
		"kind": kind.function,
		"detail": "built-in function",
		"textEdit": {"range": location.word_range(ref, position), "newText": builtin.name},
		"documentation": {"kind": "markdown", "value": template.render_for_builtin(builtin)},
	}
}
