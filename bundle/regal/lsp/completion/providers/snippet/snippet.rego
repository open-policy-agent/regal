# METADATA
# description: |
#   provides completion suggestions for **snippets** in locations where
#   it makes sense to do so. note that while snippets originally were specific
#   to VS Code, they are also supported by e.g. Zed and other editors
# related_resources:
#   - description: documentation
#     ref: https://code.visualstudio.com/docs/editor/userdefinedsnippets
package regal.lsp.completion.providers.snippet

import data.regal.lsp.completion.kind
import data.regal.lsp.completion.location

# METADATA
# description: all completion suggestions for snippets
# scope: document
items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	location.in_rule_body(line)

	word := location.word_at(line, input.params.position.character + 1)
	before := trim_suffix(line, word.text)

	# match empty line, or line ending with `if` or `|` plus whitespace
	regex.match(`^\s+$|if\s+$|\|\s+$`, before)

	some label, snippet in _snippets

	strings.any_prefix_match(snippet.prefix, word.text)
	strings.count(line, snippet.prefix[0]) < 2

	item := {
		"label": $"{label} (snippet)",
		"kind": kind.snippet,
		"detail": label,
		"textEdit": {
			"range": location.word_range(word, input.params.position),
			"newText": snippet.body,
		},
		"insertTextFormat": 2, # snippet
	}
}

items contains item if {
	line := input.regal.file.lines[input.params.position.line]

	startswith("metadata", line)
	word := location.word_at(line, input.params.position.character + 1)
	range := location.word_range(word, input.params.position)

	some item in {
		{
			"label": "metadata annotation [title, description] (snippet)",
			"kind": kind.snippet,
			"detail": "metadata annotation",
			"textEdit": {
				"range": range,
				"newText": "# METADATA\n# title: ${1:title}\n# description: ${2:description}",
			},
			"insertTextFormat": 2, # snippet
		},
		{
			"label": "metadata annotation [description] (snippet)",
			"kind": kind.snippet,
			"detail": "metadata annotation",
			"textEdit": {
				"range": range,
				"newText": "# METADATA\n# description: ${1:description}",
			},
			"insertTextFormat": 2, # snippet
		},
	}
}

_snippets := {
	"some value iteration": {
		"body": "some ${1:var} in ${2:collection}\n$0",
		"prefix": ["some"],
	},
	"some key-value iteration": {
		"body": "some ${1:key}, ${2:value} in ${3:collection}\n$0",
		"prefix": ["some", "some-kv"],
	},
	"every value iteration": {
		"body": "every ${1:var} in ${2:collection} {\n\t$0\n}",
		"prefix": ["every"],
	},
	"every key-value iteration": {
		"body": "every ${1:key}, ${2:value} in ${3:collection} {\n\t$0\n}",
		"prefix": ["every", "every-kv"],
	},
}
