# METADATA
# description: Client identifiers known by the Regal LSP server
package regal.lsp.clients

# METADATA
# title: Generic
# description: Catch-all for clients we don't know about
generic := 0

# METADATA
# title: VS Code
# description: VS Code, via the OPA extension
# related_resources:
#   - https://github.com/open-policy-agent/vscode-opa
vscode := 1

# METADATA
# title: Go Test
# description: Client identifier used in go tests for the language server (internal)
gotest := 2

# METADATA
# title: Zed
# description: The Zed Editor, via zed-rego
# related_resources:
#   - https://zed.dev/docs/languages/rego
#   - https://github.com/styraoss/zed-rego
zed := 3

# METADATA
# title: Neovim
# description: The Neovim client
# related_resources:
#   - https://neovim.io/
neovim := 4
