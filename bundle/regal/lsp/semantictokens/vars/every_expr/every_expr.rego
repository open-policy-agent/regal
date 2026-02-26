# METADATA
# description: |
#   Helper package for semantictokens that returns variable references and declarations 'every' keyword domains
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.every_expr

import data.regal.ast

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract variable declarations from every keyword domains
result.declaration contains var if {
	some rule_index in ast.rule_index_strings
	some var in ast.found.vars[rule_index]["every"]
}

# METADATA
# description: Extract variable references in every keyword domains
result.reference contains var if {
	some rule_index in ast.rule_index_strings

	declared_vars := ast.found.vars[rule_index]["every"]

	declared_var_names := {v.value | some v in declared_vars}

	walk(module.rules[to_number(rule_index)].body, [_, term])
	some var in term.terms
	var.type == "var"
	var.value in declared_var_names
	not var in declared_vars
}
