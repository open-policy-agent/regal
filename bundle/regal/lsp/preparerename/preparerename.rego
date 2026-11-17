# METADATA
# description: implementation of the LSP prepare rename feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
package regal.lsp.preparerename

import data.regal.lsp.util.range
import data.regal.lsp.util.references

# METADATA
# entrypoint: true
default result["response"] := null

result["response"] := ref.range if {
	some ref in ((references.variable_refs | references.rule_refs) | references.package_refs)

	ref.uri == input.params.textDocument.uri

	range.contains_position(ref.range, input.params.position)
}
