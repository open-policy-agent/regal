# METADATA
# description: |
#   Information about the client making the request,
#   and metadata related to language server clients.
# schemas:
#   - input:       schema.regal.lsp.common
#   - data.client: schema.regal.lsp.client
package regal.lsp.client

# METADATA
# description: |
#   The client's identifier as declared during initialization.
#   See `identifiers` below for known client identifiers.
identifier := data.client.identifier

# METADATA
# description: |
#   The client's capabilities as declared during initialization
capabilities := data.client.capabilities

# METADATA
# description: |
#   The client's initialization options as declared during initialization
init_options := data.client.init_options

# METADATA
# title: Generic
# description: Catch-all for clients we don't know about
identifiers["generic"] := 0

# METADATA
# title: VS Code
# description: VS Code, via the OPA extension
# related_resources:
#   - https://github.com/open-policy-agent/vscode-opa
identifiers["vscode"] := 1

# METADATA
# title: Go Test
# description: Client identifier used in go tests for the language server (internal)
identifiers["gotest"] := 2

# METADATA
# title: Zed
# description: The Zed Editor, via zed-rego
# related_resources:
#   - https://zed.dev/docs/languages/rego
#   - https://github.com/styraoss/zed-rego
identifiers["zed"] := 3

# METADATA
# title: Neovim
# description: The Neovim client
# related_resources:
#   - https://neovim.io/
identifiers["neovim"] := 4
