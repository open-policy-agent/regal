package regal.lsp.semantictokens

import data.regal.ast

# METADATA
# description: Extract variable declarations from some constructs
some_tokens.declaration contains var if some var in ast.found.vars[_]["some"]

# METADATA
# description: Extract variable references in some everys
some_tokens.reference contains var if {
	some rule in module.rules

	declared_var_names := {v.value | some v in ast.found.vars[_]["some"]}

	# I feel like found.vars[rule_index] should work here, but I couldn't get it working
	walk(rule.body, [_, term])
	some var in term.terms
	var.type == "var"
	var.value in declared_var_names
}
