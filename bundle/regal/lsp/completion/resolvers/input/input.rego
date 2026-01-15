# METADATA
# description: Completion resolver for the 'input' keyword's documentation
package regal.lsp.completion.resolvers.input

# METADATA
# description: provides documentation for the `input` keyword completion item
resolve := object.union(input.params, {"documentation": {
	"kind": "markdown",
	"value": `# input

'input' refers to the input document being evaluated.
It is a special keyword that allows you to access the data sent to OPA at evaluation time.

To see more examples of how to use 'input', check out the
[policy language documentation](https://www.openpolicyagent.org/docs/policy-language/).

You can also experiment with input in the [Rego Playground](https://play.openpolicyagent.org/).
`,
}})
