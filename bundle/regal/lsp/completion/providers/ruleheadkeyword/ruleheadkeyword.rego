# METADATA
# description: |
#   Returns completions in rule heads following the rule name or the end of the head.
#   Currently supported cases:
#     - [rule-name] if
#     - [rule-name] :=
#     - [rule-name] contains
#     - [rule-name] contains if
#   These completions are exclusive at their location, meaning no other provider
#   should have completions to offer here.
package regal.lsp.completion.providers.ruleheadkeyword

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: Set of suggested completion items
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	not regex.match(`^\s+`, line)
	not startswith(line, "package")
	not startswith(line, "import")

	word := location.word_at(line, input.params.position.character + 1)

	_word_matches(word.text)

	some obj in _suggestions(word.text, _words_no_space(word.text_before))

	item := object.union(obj, {"textEdit": {
		"range": location.word_range(word, input.params.position),
		"newText": concat("", [obj.label, " "]),
	}})
}

_word_matches("")
_word_matches(text) if startswith("if", text)
_word_matches(text) if startswith("contains", text)
_word_matches(text) if startswith(":=", text)

# suggest if, contains and := after rule name
_suggestions(text, words) := suggested if {
	count(words) == 1

	suggested := [completion |
		some completion in completions
		startswith(completion.label, text)
	]
}

# suggest if at the end of rule head
_suggestions(text, words) := object.filter(completions, ["if"]) if {
	count(words) == 3
	startswith("if", text)
	words[1] in {"contains", "=", ":="}
}

_words_no_space(line) := [word |
	some word in regex.split(`\s+`, line)
	word != ""
]

# TBD:
# - Add "detail" (string), "documentation" (markdown) or both attributes for each item using
#   the same data that we display on hover. this is only seen when clicking a suggestion to expand.
# - Implement the "mandatory" logic from the Go implementation.. or maybe don't, but make sure that
#   other providers don't provide suggestions in locations where they're not valid.

# METADATA
# description: The available completions this package provides
completions := {
	"if": {
		"label": "if",
		"labelDetails": {"description": "add conditions for rule to evaluate"},
		"kind": kind.keyword,
	},
	"contains": {
		"label": "contains",
		"labelDetails": {"description": "add values to multi-value rule (set)"},
		"kind": kind.keyword,
	},
	":=": {
		"label": ":=",
		"labelDetails": {"description": "assign value to rule"},
		"kind": kind.operator,
	},
}
