# METADATA
# description: |
#   Resolver that renders markdown tooltips for inlay hints on demand, to
#   avoid rendering hundreds of tooltips that most often won't be seen by the user
# schemas:
#   - input: schema.regal.lsp.common
#   - input.params: {type: object}
package regal.lsp.inlayhint_resolve

# METADATA
# entrypoint: true
# description: |
#   Resolve inlay hint tooltip information on demand using argument
#   `data` passed from the initial inlay hint response
# scope: document
result["response"] := resolved if {
	tooltip := object.union(input.params, {"tooltip": {
		"kind": "markdown",
		"value": markdown(input.params.data),
	}})

	resolved := object.remove(tooltip, ["data"])
}

# METADATA
# description: no data = nothing to resolve, return params as is
# scope: rule
result["response"] := input.params if not input.params.data

# METADATA
# description: |
#   Render markdown tooltip for inlay hint using argument data
# scope: document
markdown(info) := $"`{info.name}` — {info.type}" if not info.description
markdown(info) := $"`{info.name}` — {info.type}: {info.description}" if info.description
