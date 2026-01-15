# METADATA
# description: |
#   Completion resolvers for selected completion items. These help provide additional informtion, like
#   documentation, at the time requested by the client rather than up-front together with the completion items.
#   Less data to process and transfer up-front, means faster time until completion suggestions are shown to the user.
# scope: subpackages
# schemas:
#   # override for resolvers, as params contain the original completion item
#   - input.params: {}
package regal.lsp.completion.resolvers
