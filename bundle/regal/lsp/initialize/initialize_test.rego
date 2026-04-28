package regal.lsp.initialize_test

import data.regal.lsp.initialize

test_initialize_experimental_capabilities if {
	response := initialize.result.response
		with input.params as {
			"capabilities": {},
			"initializationOptions": {},
			"clientInfo": {"name": "go test"},
		}
		with input.regal.server.feature_flags as {
			"explorer_provider": true,
			"inline_evaluation_provider": true,
			"debug_provider": true,
			"opa_test_provider": true,
		}

	response.capabilities.experimental.explorerProvider == true
	response.capabilities.experimental.inlineEvalProvider == true
	response.capabilities.experimental.debugProvider == true
	response.capabilities.experimental.opaTestProvider == true
}

test_initialize_client_identifier[client_name] if {
	some identifier, client_name in [
		"Space Editor!!1!",
		"Visual Studio Code",
		"go test",
		"Zed",
		"Neovim",
		"IntelliJ IDEA 2027.1",
	]

	result := initialize.result
		with input.params.clientInfo.name as client_name

	result.regal.client.identifier == identifier
}

test_initialize_client_capabilities if {
	result := initialize.result
		with input.params.capabilities.textDocument.inlayHint.resolveSupport.properties as ["tooltip"]

	result.regal.client.capabilities.textDocument.inlayHint.resolveSupport.properties == ["tooltip"]
}
