# METADATA
# description: |
#   base package for completion suggestion provider policies, and acts
#   like a router that collects suggestions from all provider policies
#   under regal.lsp.completion.providers
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.completion
# scope: subpackages
package regal.lsp.completion

import data.regal.lsp.client
import data.regal.lsp.location

import data.regal.util

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := {
	"items": items,
	"isIncomplete": true,
} if {
	input.method == "textDocument/completion"
	not _default_edit_range_supported
}

# Client supports setting a default edit range for completion items.
result["response"] := {
	"items": items,
	"isIncomplete": false,
	"itemDefaults": {"editRange": range},
} if {
	input.method == "textDocument/completion"
	_default_edit_range_supported

	line := input.regal.file.lines[input.params.position.line]
	line != ""

	location.in_rule_body(line)

	ref := location.ref_at(line, input.params.position.character + 1)
	range := location.word_range(ref, input.params.position)
}

# METADATA
# schemas:
#   - input: {}
result["response"] := data.regal.lsp.completion.resolvers[input.params.data].resolve if {
	input.method == "completionItem/resolve"
} else := input.params if {
	# if there was nothing to resolve, return the input as-is
	input.method == "completionItem/resolve"
}

# METADATA
# description: main entry point for completion suggestions
# entrypoint: true
items contains completion if {
	# exit early if caret position is inside a comment. We currently don't have any provider
	# where doing completions inside of a comment makes sense. Behavior is also editor-specific:
	# - Zed: always on, with no way to disable
	# - VSCode: disabled but can be enabled with "editor.quickSuggestions.comments" setting
	not inside_comment

	some provider, completion
	data.regal.lsp.completion.providers[provider].items[completion]
}

# METADATA
# description: |
#   checks if the current position is inside a comment
inside_comment if {
	# avoid unmarshalling every comment location but only one that starts
	# with the line number of the current position
	line := $"{input.params.position.line + 1}:"

	some comment in data.workspace.parsed[input.params.textDocument.uri].comments

	startswith(comment.location, line)
	util.to_location_no_text(comment.location).col <= input.params.position.character + 1
}

_default_edit_range_supported if {
	"editRange" in client.capabilities.textDocument.completion.completionList.itemDefaults
}
