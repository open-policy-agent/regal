# METADATA
# description: |
#   Reports links found in provided document. Most editors will treat HTTP URL's
#   as links automatically, so there's no need to report those. We can however link to
#   other documents in the workspace when appropriate. Potential applications:
#     - Rule names in inline ignore directives link to rule docs
#     - Refs in schema annotations link to their schema file
#     - Identifiers in doc comments enclosed in brackets link to definition (like Go)
#     - Imports link to their package (why not done by goto definition?)
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentLink
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.documentlink

import data.regal.ast

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := items if items != set()

# METADATA
# description: Set of links in document
# entrypoint: true
items contains item if {
	some location in ast.comments_decoded

	contains(location.text, "regal ignore:")

	row := location.row - 1

	some rule in regex.split(`,\s*`, trim_space(regex.replace(location.text, `^.*regal ignore:\s*(\S+)`, "$1")))

	col := (location.col + indexof(location.text, rule)) - 1

	item := {
		"target": $"https://www.openpolicyagent.org/projects/regal/rules/{_category_for[rule]}/{rule}",
		"range": {
			"start": {
				"line": row,
				"character": col,
			},
			"end": {
				"line": row,
				"character": col + count(rule),
			},
		},
		"tooltip": $"See documentation for {rule}",
	}
}

_category_for[rule] := category if {
	some category, rules in data.workspace.config.rules
	some rule, _ in rules
}
