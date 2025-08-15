# METADATA
# description: |
#   base package for completion suggestion provider policies, and acts
#   like a router that collects suggestions from all provider policies
#   under regal.lsp.completion.providers
package regal.lsp.completion

# METADATA
# entrypoint: true
result["response"] := items

# METADATA
# description: main entry point for completion suggestions
# entrypoint: true
items contains object.union(completion, {"_regal": {"provider": provider}}) if {
	some provider, completion
	data.regal.lsp.completion.providers[provider].items[completion]
}
