# METADATA
# description: |
#   Linked editing ranges allow having a local rename *on type* reflect in multiple places. Provides the ranges
#   of the following, and of their references in the file:
#     - function arguments
#     - variables declared with :=, some, every and in comprehensions
#     - rules and functions that are only declared and used in the current file
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_linkedEditingRange
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocumentposition
package regal.lsp.linkededitingrange

import data.regal.lsp.util.references

# METADATA
# entrypoint: true
result.response.ranges := ranges

# METADATA
# description: Link the variable or function argument in position to its references in the file
ranges contains ref.range if {
	some ref in references.variable_refs

	ref.uri == input.params.textDocument.uri
}

# METADATA
# description: |
#   Link the rule or function in position to its heads and references, but only when they are all in this file,
#   as the other files can't be edited at the same time
ranges contains ref.range if {
	{ref.uri | some ref in references.rule_refs} == {input.params.textDocument.uri}

	some ref in references.rule_refs
}
