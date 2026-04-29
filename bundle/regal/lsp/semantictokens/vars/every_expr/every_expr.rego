# METADATA
# description: |
#   Helper package for semantictokens that returns variable references and declarations 'every' keyword domains
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.textdocument
package regal.lsp.semantictokens.vars.every_expr

import data.regal.ast
import data.regal.util

# METADATA
# description: Get the module from workspace
module := data.workspace.parsed[input.params.textDocument.uri]

# METADATA
# description: Extract variable definitions from every keyword domains
result contains token if {
	some rule_index
	declared_vars := ast.found.vars[rule_index]["every"]
	some var in declared_vars

	tloc := util.to_location_object(var.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 1,
		"modifiers": bits.lsh(1, 1),
	}
}

# METADATA
# description: Extract variable references in every keyword domains
result contains token if {
	some rule_index

	declared_vars := ast.found.vars[rule_index]["every"]

	declared_var_names := {v.value | some v in declared_vars}

	some every_terms in ast.found.every[rule_index]
	walk(every_terms.body, [_, expr])

	some var in expr.terms
	var.type == "var"
	var.value in declared_var_names
	not var in declared_vars

	tloc := util.to_location_object(var.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 1,
		"modifiers": bits.lsh(1, 2),
	}
}
