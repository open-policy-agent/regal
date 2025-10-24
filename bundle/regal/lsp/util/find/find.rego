# METADATA
# description: Common functions for finding AST elements at a given position
# schemas:
#   - input: schema.regal.lsp.common
package regal.lsp.util.find

import data.regal.lsp.completion.location
import data.regal.lsp.util.location as uloc

# METADATA
# description: find the function argument at the given position, if any
# schemas:
#   - input.params: schema.regal.lsp.textdocumentposition
arg_at_position := [arg, rule_index] if {
	text := input.regal.file.lines[input.params.position.line]
	word := location.word_at(text, input.params.position.character)

	some rule_index
	arg := data.workspace.parsed[input.params.textDocument.uri].rules[rule_index].head.args[_]

	arg.type == "var"
	arg.value == word.text

	arg_pos := uloc.parse_range(arg.location)

	input.params.position.line == arg_pos.start.line
	input.params.position.character >= arg_pos.start.character
	input.params.position.character <= arg_pos.end.character
}

# METADATA
# description: find the rule at input.params.position if any
# schemas:
#   - input.params: schema.regal.lsp.textdocumentposition
rule_at_position := [rule, rule_index] if {
	some rule_index, rule in data.workspace.parsed[input.params.textDocument.uri].rules

	uloc.within_range(input.params.position, uloc.parse_range(rule.location))
}

# METADATA
# description: find the import statement at input.params.position, if any
# schemas:
#   - input.params: schema.regal.lsp.textdocumentposition
import_at_position := imp if {
	some imp in data.workspace.parsed[input.params.textDocument.uri].imports

	uloc.within_range(input.params.position, uloc.parse_range(imp.path.location))
}
