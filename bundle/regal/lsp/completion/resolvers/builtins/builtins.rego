# METADATA
# description: Completion resolver for built-in function documentation
package regal.lsp.completion.resolvers.builtins

import data.regal.lsp.template

# METADATA
# description: Provides documentation for built-in functions
# schemas:
resolve := object.union(input.params, {"documentation": {
	"kind": "markdown",
	"value": template.render_for_builtin(data.workspace.builtins[input.params.label]),
}})
