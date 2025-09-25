# METADATA
# description: |
#   The rulename provider returns completions at the start of a line, suggesting
#   names of other rules in the package
package regal.lsp.completion.providers.rulename

import data.regal.ast

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: Set of suggested names
items contains item if {
	count(input.regal.file.lines) > 1

	line := input.regal.file.lines[input.params.position.line]
	word := location.word_at(line, input.params.position.character + 1)

	not regex.match(`\s`, word.text_before)

	rules := {[name, _rule_kind(rule)] |
		some rule in data.workspace.parsed[input.params.textDocument.uri].rules
		name := ast.ref_static_to_string(rule.head.ref)
		not startswith(name, "test_")
	}

	some [name, kind] in rules

	startswith(name, word.text)

	item := {
		"label": name,
		"kind": kind,
		"detail": _kind_detail[kind],
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": concat("", [name, " "]),
		},
	}
}

_kind_detail := {
	kind.variable: "rule",
	kind.constant: "rule (constant)",
	kind.function: "function",
}

default _rule_kind(_) := 6 # kind.variable, but can't be referenced from default rule

_rule_kind(rule) := kind.constant if {
	not rule.default
	not rule.body
	not rule.head.args
}

_rule_kind(rule) := kind.function if rule.head.args
