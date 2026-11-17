# METADATA
# description: implementation of the LSP references feature
# schemas:
#   - input:                schema.regal.lsp.common
#   - input.params:         schema.regal.lsp.textdocumentposition
#   - input.params.context: {type: "object", properties: {includeDeclaration: {type: "boolean"}}}
package regal.lsp.references

import data.regal.lsp.util.references

# METADATA
# entrypoint: true
result.response contains object.remove(ref, ["kind"]) if some ref in references.refs
