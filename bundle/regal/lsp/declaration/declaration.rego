# METADATA
# description: |
#   Declaration handler points to the location of the symbol (e.g. var in `some var`)
#   the variable at the pointer was declared, if such a declaration exists.
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_declaration
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.declaration

import future.keywords.or

import data.regal.ast
import data.regal.lsp.client
import data.regal.lsp.location as lsp_location
import data.regal.lsp.util.range

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := symbol_location

# METADATA
# description: |
#   Returns the locationLink of the symbol declaration if it exists
#   and the client supports that format.
symbol_location := location_link if {
	client.capabilities.textDocument.declaration.linkSupport == true

	[rule_range, decl_range] := _rule_and_decl_ranges

	location_link := [{
		"targetUri": input.params.textDocument.uri,
		"targetRange": rule_range,
		"targetSelectionRange": decl_range,
	}]
}

# METADATA
# description: |
#   Returns the location of the symbol declaration if link support is not available.
symbol_location := location if {
	not client.capabilities.textDocument.declaration.linkSupport

	location := {
		"uri": input.params.textDocument.uri,
		"range": _rule_and_decl_ranges[1],
	}
}

_rule_and_decl_ranges := decls[0] if {
	line := input.params.position.line
	text := input.regal.file.lines[line]
	word := lsp_location.word_at(text, input.params.position.character + 1)

	rules := data.workspace.parsed[input.params.textDocument.uri].rules
	index := lsp_location.find_rule(rules, line + 1)
	decls := [[rule_range, decl_range] |
		some term in ast.found.vars[index].some
		term.value == word.text

		rule_range := range.parse(rules[index].location)
		decl_range := range.parse(term.location)

		decl_range.start.line < line or line == rule_range.start.line
	]
}

_link_support := client.capabilities.textDocument.declaration.linkSupport
