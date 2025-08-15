# METADATA
# description: |
#   Main entrypoint for the parts of the language server implemented in Rego
# schemas:
#   - input: schema.regal.lsp.common
package regal.lsp.main

# TBD:
#   - Runtime schema validation of inputs and outputs would be great to
#     do here as we'd leave that out of the handlers

# METADATA
# entrypoint: true
eval := response if {
	handler := _handler_for(input.method)
	result := data.regal.lsp[handler].result

	response := {
		# The payload as defined in the language server specification
		"response": result.response,
		# Optional metadata that handlers may want to send back as part
		# of their response. This can be processed either here or in the
		# Go handler later.
		"regal": object.get(result, ["regal"], {}),
	}
}

_handler_for(method) := lower(name) if ["textDocument", name] = split(method, "/")
