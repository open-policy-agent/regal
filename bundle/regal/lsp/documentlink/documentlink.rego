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
#   - input.params: schema.regal.lsp.documentlink
package regal.lsp.documentlink

import data.regal.util

# METADATA
# entrypoint: true
result["response"] := items

# METADATA
# description: Set of links in document
# entrypoint: true
items contains item if {
	module := data.workspace.parsed[input.params.textDocument.uri]

	some encoded in module.comments
	comment := object.union(encoded, {"text": base64.decode(encoded.text)})
	contains(comment.text, "regal ignore:")

	loc := util.to_location_no_text(comment.location)
	rules := regex.split(`,\s*`, trim_space(regex.replace(comment.text, `^.*regal ignore:\s*(\S+)`, "$1")))

	some rule in rules

	pos := indexof(comment.text, rule)
	row := loc.row - 1
	col := loc.col + pos

	item := {
		"target": concat("/", ["https://docs.styra.com/regal/rules", _category_for[rule], rule]),
		"range": {
			"start": {"line": row, "character": col},
			"end": {"line": row, "character": col + count(rule)},
		},
		"tooltip": concat(" ", ["See documentation for", rule]),
	}
}

_category_for[rule] := category if {
	some category, rules in data.workspace.config.rules
	some rule, _ in rules
}
