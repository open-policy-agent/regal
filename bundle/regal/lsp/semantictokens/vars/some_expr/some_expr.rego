# METADATA
# description: |
#   Helper package for semantictokens that returns variable references and declarations in 'every' keyword domains
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.some_expr

import data.regal.ast

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract variable declarations from some keyword domains
result.declaration contains var if {
	some context in ["somein", "some"]
	some rule_index in ast.rule_index_strings
	some var in ast.found.vars[rule_index][context]
}

# METADATA
# description: Extract variable references in some keyword domains
result.reference contains var if {
	some rule_index in ast.rule_index_strings
	some context in ["somein", "some"]

	declared_var_names := {v.value | some v in ast.found.vars[rule_index][context]}

	walk(module.rules[to_number(rule_index)].body, [_, term])
	some var in term.terms
	var.type == "var"
	var.value in declared_var_names
}
