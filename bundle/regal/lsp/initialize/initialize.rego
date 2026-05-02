# METADATA
# description: |
#   handler for the LSP "initialize" request, which is the first request sent by the client to the server in order
#   to initialize the connection and establish the capabilities of both client and server
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.initialize
# scope: subpackages
package regal.lsp.initialize

# METADATA
# entrypoint: true
result.response.serverInfo := _serverInfo

_serverInfo.name := "Regal"

default _serverInfo.version := "unknown"

_serverInfo.version := data.server.version

# METADATA
# description: The capabilities of Regal's language server, as defined in the LSP specification
result.response.capabilities := _capabilities

_capabilities.textDocumentSync := {
	"openClose": true,
	# For now, send full document on change, but this is something we should improve.
	# See https://github.com/open-policy-agent/regal/issues/1651
	"change": 1,
	"save": {"includeText": true},
}

_capabilities.diagnosticProvider := {
	"identifier": "rego",
	"interFileDependencies": true,
	"workspaceDiagnostics": true,
}

_capabilities.workspace.fileOperations[operation].filters := filters if {
	filters := [{
		"scheme": "file",
		"pattern": {"glob": "**/*.rego"},
	}]
	some operation in ["didCreate", "didRename", "didDelete"]
}

## NOTE(anders): The language server protocol doesn't go into detail about what this is meant to
## entail, and there's nothing else in the request/response payloads that carry workspace folder
## information. The best source I've found on the this topic is this example repo from VS Code,
## where they have the client start one instance of the server per workspace folder:
## https://github.com/microsoft/vscode-extension-samples/tree/main/lsp-multi-server-sample
## That seems like a reasonable approach to take, and means we won't have to deal with workspace
## folders throughout the rest of the codebase. But the question then is — what is the point of
## this capability, and what does it mean to say we support it? Clearly we don't in the server as
## *there is no way* to support it here.
_capabilities.workspace.workspaceFolders.supported := true

_capabilities.inlayHintProvider := {
	# inlayHint/resolve request supported
	"resolveProvider": true,
}

_capabilities.hoverProvider := true

_capabilities.signatureHelpProvider := {
	# In additional to the client's default trigger characters for signature help
	"triggerCharacters": ["(", ","]
}

_capabilities.codeActionProvider := {
	# Currently supported code action kinds
	"codeActionKinds": [
		"quickfix",
		"source"
	],
}

_capabilities.executeCommandProvider.commands := _commands

_commands contains "regal.eval"
_commands contains "regal.fix.opa-fmt"
_commands contains "regal.fix.use-rego-v1"
_commands contains "regal.fix.use-assignment-operator"
_commands contains "regal.fix.no-whitespace-comment"
_commands contains "regal.fix.directory-package-mismatch"
_commands contains "regal.fix.non-raw-regex-pattern"
_commands contains "regal.fix.prefer-equals-comparison"
_commands contains "regal.fix.constant-condition"
_commands contains "regal.fix.redundant-existence-check"
_commands contains "regal.config.disable-rule"
_commands contains "regal.explorer" if data.server.feature_flags.explorer_provider
_commands contains "regal.debug" if data.server.feature_flags.debug_provider

_capabilities.documentFormattingProvider := true

_capabilities.foldingRangeProvider := true

_capabilities.definitionProvider := true

_capabilities.documentSymbolProvider := true

_capabilities.workspaceSymbolProvider := true

_capabilities.completionProvider := {
	"triggerCharacters": [
		":", # to suggest :=
		".", # for refs
	],
	"resolveProvider": true,
	"completionItem": {"labelDetailsSupport": true},
}

_capabilities.codeLensProvider := {
	# codeLens/resolve to be implemented
	"resolveProvider": false,
}

_capabilities.documentLinkProvider := {
	# documentLink/resolve to be implemented
	"resolveProvider": false,
}

_capabilities.documentHighlightProvider := true

_capabilities.selectionRangeProvider := true

_capabilities.linkedEditingRangeProvider := true

_capabilities.semanticTokensProvider := {
	"legend": {
		"tokenTypes": [
			"namespace",
			"variable",
			"import",
			"keyword",
		],
		"tokenModifiers": [
			"declaration",
			"definition",
			"reference",
		],
	},
	"full": true,
}

# ExperimentalCapabilities contains Regal-specific custom LSP features
# that are not part of the base LSP specification. 'Experimental' comes
# from the field name in the spec, rather than their status. 'Experimental'
# features are more like 'custom' features we have built on the LSP.

# METADATA
# description: |
#   explorerProvider indicates whether the server supports the regal.explorer
#   command and the regal/showExplorerResult notification.
_capabilities.experimental.explorerProvider := data.server.feature_flags.explorer_provider

# METADATA
# description: |
#   inlineEvalProvider indicates whether the server supports the regal.eval
#   command response being sent rather than written to file.
_capabilities.experimental.inlineEvalProvider := data.server.feature_flags.inline_evaluation_provider

# METADATA
# description: |
#   debugProvider indicates whether the server supports the regal.debug
#   command and regal/startDebugging request.
_capabilities.experimental.debugProvider := data.server.feature_flags.debug_provider

# METADATA
# description: |
#   opaTestProvider indicates whether the server supports testing-related features
#   including running Rego tests via LSP command and test location notifications.
_capabilities.experimental.opaTestProvider := data.server.feature_flags.opa_test_provider

# METADATA
# description: The server's identifier for the client, based on the clientInfo sent in the request
# scope: document
default result.regal.client.identifier := 0

result.regal.client.identifier := _client_identifier(input.params.clientInfo.name)

_client_identifier("Visual Studio Code") := 1
_client_identifier("go test") := 2
_client_identifier("Zed") := 3
_client_identifier("Neovim") := 4
_client_identifier(name) := 5 if contains(name, "IntelliJ")

# METADATA
# description: The initialization options sent by the client, or an empty object if not provided
# scope: document
default result.regal.client.initializationOptions := {}

result.regal.client.initializationOptions := input.params.initializationOptions

# METADATA
# description: The capabilities of the client, as sent in the initialize request
# scope: document
result.regal.client.capabilities := input.params.capabilities

# METADATA
# description: The root URI of the workspace, as provided by the client
result.regal.workspace.uri := input.params.rootUri if {
	input.params.rootUri != ""
} else := input.params.workspaceFolders[0].uri

# METADATA
# description: Any warnings to log from initialization
# scope: document
result.regal.warnings contains $"multiple workspace folders provided, only the first one will be used: {uri}" if {
	count(input.params.workspaceFolders) > 1

	uri := input.params.workspaceFolders[0].uri
}
