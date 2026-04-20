# METADATA
# description: |
#   Helper package for semantictokens that returns variable references and declarations in comprehensions
# schemas:
#   - input:        schema.regal.lsp.common
#   - input.params: schema.regal.lsp.semantictokens
package regal.lsp.semantictokens.vars.comprehensions

import data.regal.ast
import data.regal.util

_declared_var_names contains name if {
	comprehension := ast.found.comprehensions[_][_]

	some expr in comprehension.value.body
	some symbol in expr.terms.symbols
	some var in array.slice(symbol.value, 1, count(symbol.value) - 1)
	var.type == "var"
	name := var.value
}

# METADATA
# description: Extract comprehension variable declarations from array/set/object comprehensions
result contains token if {
	comprehension := ast.found.comprehensions[_][_]

	some expr in comprehension.value.body
	some symbol in expr.terms.symbols
	some var in array.slice(symbol.value, 1, count(symbol.value) - 1)
	var.type == "var"

	tloc := util.to_location_object(var.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 1,
		"modifiers": bits.lsh(1, 0),
	}
}

# METADATA
# description: Extract comprehension variable references in the output
result contains token if {
	comprehension := ast.found.comprehensions[_][_]

	output_vars := array.flatten([
		_comprehension_key(comprehension),
		_comprehension_value(comprehension),
		_comprehension_body_vars(comprehension.value.body),
	])
	some var in output_vars
	var.type == "var"
	var.value in _declared_var_names

	tloc := util.to_location_object(var.location)

	token := {
		"line": tloc.row - 1,
		"col": tloc.col - 1,
		"length": tloc.end.col - tloc.col,
		"type": 1,
		"modifiers": bits.lsh(1, 1),
	}
}

default _comprehension_value(_) := set()

# Helper to get variables from differing comprehensions
_comprehension_value(comprehension) := value if {
	comprehension.type in ["arraycomprehension", "setcomprehension"]
	value := comprehension.value.term
} else := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.value
}

default _comprehension_key(_) := set()

# Helper to get variables from differing comprehensions
_comprehension_key(comprehension) := value if {
	comprehension.type == "objectcomprehension"
	value := comprehension.value.key
}

# Helper to get variables from differing comprehensions
_comprehension_body_vars(body) := [value |
	some expr in body
	not expr.terms.symbols
	some value in expr.terms
	value.type == "var"
]
