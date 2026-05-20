# METADATA
# description: |
#   This module contains the logic for handling parts of the 'initialized'
#   notification from the client. The response from this module will be passed
#   to the response handler for the 'initialized' notification, which does some
#   additional work necessarily done in Go. The Rego part of the handler currently
#   does nothing but conditionally register additional workspace/didChangeWatchedFiles
#   watchers if the client supports it. Hopefully we can expand that in the future.
# schemas:
#   - input: schema.regal.lsp.common
package regal.lsp.initialized

import data.regal.lsp.client

# METADATA
# entrypoint: true
default result["response"] := null

# METADATA
# description:
result["response"] := {"registrations": [{
	"id": "regal-watched-files",
	"method": "workspace/didChangeWatchedFiles",
	"registerOptions": {"watchers": [
		{"globPattern": "**/*.rego"},
		{"globPattern": "**/*.json"},
		{"globPattern": "**/*.yaml"},
	]},
}]} if {
	client.capabilities.workspace.didChangeWatchedFiles.dynamicRegistration
}
