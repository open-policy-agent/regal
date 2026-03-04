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
	some context in {"somein", "some"}
	some rule_index

	# regal ignore:prefer-some-in-iteration
	declared_vars := ast.found.vars[rule_index][context]

	some var in declared_vars
}

# METADATA
# description: Extract variable references in some keyword domains
result.reference contains var if {
	some rule_index, startpoint in _some_start_points
	some context in {"somein", "some"}

	# regal ignore:prefer-some-in-iteration
	declared_vars := ast.found.vars[rule_index][context]

	first_some_row := min(object.keys(startpoint))

	some row, expr
	_rule_exprs[rule_index][row][expr]
	row > first_some_row

	declared_var_names := {v.value | some v in declared_vars}

	term_vars := ast.find_term_vars(expr)
	some var in term_vars

	var.type == "var"
	var.value in declared_var_names
	not var in declared_vars
}

_some_start_points[rule_index][row] contains some_var if {
	some rule_index
	some context in {"somein", "some"}

	# regal ignore:prefer-some-in-iteration
	declared_vars := ast.found.vars[rule_index][context]

	some some_var in declared_vars
	row := to_number(substring(
		some_var.location, 0,
		indexof(some_var.location, ":"),
	))
}

_rule_exprs[rule_index][row] contains expr if {
	some i
	expr := module.rules[i].body[_]
	rule_index := ast.rule_index_strings[i]
	row := to_number(substring(expr.location, 0, indexof(
		expr.location,
		":",
	)))
}
