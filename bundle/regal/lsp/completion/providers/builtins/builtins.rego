# METADATA
# description: Completion suggestions for built-in functions
package regal.lsp.completion.providers.builtins

import data.regal.lsp.client
import data.regal.lsp.completion.kind
import data.regal.lsp.location

# METADATA
# description: suggest built-in functions matching typed ref
# scope: document

# METADATA
# description: without editRange item defaults (large amount of duplicated data)
items contains item if {
	not _default_edit_range_supported

	line := input.regal.file.lines[input.params.position.line]

	line != ""
	not startswith(line, "default ")
	location.in_rule_body(line)

	ref := location.ref_at(line, input.params.position.character + 1)

	some builtin in data.workspace.builtins

	not builtin.infix # avoid suggesting 'eq', 'plus', etc
	not builtin.deprecated
	not startswith(builtin.name, "internal.")

	startswith(builtin.name, ref.text)

	item := {
		"label": builtin.name,
		"kind": kind.function,
		"detail": "built-in function",
		"textEdit": {"range": location.word_range(ref, input.params.position), "newText": builtin.name},
		"data": "builtins",
	}
}

# METADATA
# description: with editRange item defaults (range defined in main, label used as textEdit.newText))
items contains item if {
	_default_edit_range_supported

	line := input.regal.file.lines[input.params.position.line]

	not startswith(line, "default ")

	ref := location.ref_at(line, input.params.position.character + 1)

	some builtin in data.workspace.builtins

	not builtin.infix # avoid suggesting 'eq', 'plus', etc
	not builtin.deprecated
	not startswith(builtin.name, "internal.")

	startswith(builtin.name, ref.text)

	item := {
		"label": builtin.name,
		"kind": kind.function,
		"detail": "built-in function",
		"data": "builtins",
	}
}

_default_edit_range_supported if {
	"editRange" in client.capabilities.textDocument.completion.completionList.itemDefaults
}
