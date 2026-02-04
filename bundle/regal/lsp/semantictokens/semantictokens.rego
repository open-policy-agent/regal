# METADATA
# description: |
#   Returns location of variables to be highlighted via semantic tokens. Currently returns:
#     - declarations of function args in text documents
#     - variable references that are used in function calls
#     - variable references that are used in expressions
# related_resources:
#   - https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
package regal.lsp.semantictokens

import data.regal.ast

# METADATA
# description: finds declarations of function args in text documents
arg_tokens[var] := "declaration" if {
	some rule_index, contexts in ast.found.vars
	some var in contexts.args
}

# METADATA
# description: finds variable references that are used in function calls
arg_tokens[var] := "reference" if {
	some rule_index, calls in ast.function_calls
	some call in calls
	some var in call.args
	var.type == "var"

	arg_names := {v.value | some v in ast.found.vars[rule_index].args}
	var.value in arg_names
}

# METADATA
# description: finds variable references that are used in expressions
arg_tokens[var] := "reference" if {
	some rule_index, expressions in ast.found.expressions
	some expr in expressions
	some var in expr.terms
	var.type == "var"

	arg_names := {v.value | some v in ast.found.vars[rule_index].args}
	var.value in arg_names
}
